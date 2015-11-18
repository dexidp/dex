package user

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/coreos/go-oidc/jose"
	"github.com/coreos/go-oidc/key"
	"github.com/coreos/go-oidc/oidc"

	"github.com/coreos/dex/repo"
)

const (
	bcryptHashCost = 10

	// Blowfish, the algorithm underlying bcrypt, has a maximum
	// password length of 72. We explicitly track and check this
	// since the bcrypt library will silently ignore portions of
	// a password past the first 72 characters.
	maxSecretLength = 72
)

var (
	PasswordHasher = DefaultPasswordHasher

	ErrorInvalidPassword     = errors.New("invalid Password")
	ErrorPasswordHashNoMatch = errors.New("password and hash don't match")
	ErrorPasswordExpired     = errors.New("password has expired")
)

type Hasher func(string) ([]byte, error)

func DefaultPasswordHasher(s string) ([]byte, error) {
	pwHash, err := bcrypt.GenerateFromPassword([]byte(s), bcryptHashCost)
	if err != nil {
		return nil, err
	}
	return Password(pwHash), nil
}

type Password []byte

func NewPasswordFromPlaintext(plaintext string) (Password, error) {
	return PasswordHasher(plaintext)
}

type PasswordInfo struct {
	UserID string

	Password Password

	PasswordExpires time.Time
}

func (p PasswordInfo) Authenticate(plaintext string) (*oidc.Identity, error) {
	if err := bcrypt.CompareHashAndPassword(p.Password, []byte(plaintext)); err != nil {
		return nil, ErrorPasswordHashNoMatch
	}

	if !p.PasswordExpires.IsZero() && time.Now().After(p.PasswordExpires) {
		return nil, ErrorPasswordExpired
	}

	ident := p.Identity()
	return &ident, nil
}

func (p PasswordInfo) Identity() oidc.Identity {
	return oidc.Identity{
		ID: p.UserID,
	}
}

type PasswordInfoRepo interface {
	Get(tx repo.Transaction, id string) (PasswordInfo, error)
	Update(repo.Transaction, PasswordInfo) error
	Create(repo.Transaction, PasswordInfo) error
}

func NewPasswordInfoRepo() PasswordInfoRepo {
	return &memPasswordInfoRepo{
		pws: make(map[string]PasswordInfo),
	}
}

type memPasswordInfoRepo struct {
	pws map[string]PasswordInfo
}

func (m *memPasswordInfoRepo) Get(_ repo.Transaction, id string) (PasswordInfo, error) {
	pw, ok := m.pws[id]
	if !ok {
		return PasswordInfo{}, ErrorNotFound
	}
	return pw, nil
}

func (m *memPasswordInfoRepo) Create(_ repo.Transaction, pw PasswordInfo) error {
	_, ok := m.pws[pw.UserID]
	if ok {
		return ErrorDuplicateID
	}

	if pw.UserID == "" {
		return ErrorInvalidID
	}

	if len(pw.Password) == 0 {
		return ErrorInvalidPassword
	}

	m.pws[pw.UserID] = pw
	return nil
}

func (m *memPasswordInfoRepo) Update(_ repo.Transaction, pw PasswordInfo) error {
	if pw.UserID == "" {
		return ErrorInvalidID
	}

	_, ok := m.pws[pw.UserID]
	if !ok {
		return ErrorNotFound
	}

	if len(pw.Password) == 0 {
		return ErrorInvalidPassword
	}

	m.pws[pw.UserID] = pw
	return nil
}

func (u *PasswordInfo) UnmarshalJSON(data []byte) error {
	var dec struct {
		UserID            string    `json:"userId"`
		PasswordHash      string    `json:"passwordHash"`
		PasswordPlaintext string    `json:"passwordPlaintext"`
		PasswordExpires   time.Time `json:"passwordExpires"`
	}

	err := json.Unmarshal(data, &dec)
	if err != nil {
		return fmt.Errorf("invalid User entry: %v", err)
	}

	u.UserID = dec.UserID

	u.PasswordExpires = dec.PasswordExpires

	if len(dec.PasswordHash) != 0 {
		if dec.PasswordPlaintext != "" {
			return ErrorInvalidPassword
		}
		u.Password = Password(dec.PasswordHash)
		return nil
	}
	if dec.PasswordPlaintext != "" {
		u.Password, err = NewPasswordFromPlaintext(dec.PasswordPlaintext)
		if err != nil {
			return err
		}
	}
	return nil
}

func newPasswordInfosFromReader(r io.Reader) ([]PasswordInfo, error) {
	var pws []PasswordInfo
	err := json.NewDecoder(r).Decode(&pws)
	return pws, err
}

func readPasswordInfosFromFile(loc string) ([]PasswordInfo, error) {
	pwf, err := os.Open(loc)
	if err != nil {
		return nil, fmt.Errorf("unable to read password info from file %q: %v", loc, err)
	}

	return newPasswordInfosFromReader(pwf)
}

func LoadPasswordInfos(repo PasswordInfoRepo, pws []PasswordInfo) error {
	for i, pw := range pws {
		err := repo.Create(nil, pw)
		if err != nil {
			return fmt.Errorf("error loading PasswordInfo[%d]: %q", i, err)
		}
	}
	return nil
}

func NewPasswordInfoRepoFromPasswordInfos(pws []PasswordInfo) PasswordInfoRepo {
	memRepo := NewPasswordInfoRepo().(*memPasswordInfoRepo)
	for _, pw := range pws {
		memRepo.pws[pw.UserID] = pw
	}
	return memRepo
}

func NewPasswordInfoRepoFromFile(loc string) (PasswordInfoRepo, error) {
	pws, err := readPasswordInfosFromFile(loc)
	if err != nil {
		return nil, err
	}

	return NewPasswordInfoRepoFromPasswordInfos(pws), nil
}

func NewPasswordReset(userID string, password Password, issuer url.URL, clientID string, callback url.URL, expires time.Duration) PasswordReset {
	claims := oidc.NewClaims(issuer.String(), userID, clientID, clock.Now(), clock.Now().Add(expires))
	claims.Add(ClaimPasswordResetPassword, string(password))
	claims.Add(ClaimPasswordResetCallback, callback.String())
	return PasswordReset{claims}
}

type PasswordReset struct {
	Claims jose.Claims
}

// ParseAndVerifyPasswordResetToken parses a string into a an
// PasswordReset, verifies the signature, and ensures that required
// claims are present.  In addition to the usual claims required by
// the OIDC spec, "aud" and "sub" must be present as well as
// ClaimPasswordResetCallback and ClaimPasswordResetPassword.
func ParseAndVerifyPasswordResetToken(token string, issuer url.URL, keys []key.PublicKey) (PasswordReset, error) {
	tokenClaims, err := parseAndVerifyTokenClaims(token, issuer, keys)
	if err != nil {
		return PasswordReset{}, err
	}

	pw, ok, err := tokenClaims.Claims.StringClaim(ClaimPasswordResetPassword)
	if err != nil {
		return PasswordReset{}, err
	}
	if !ok || pw == "" {
		return PasswordReset{}, fmt.Errorf("no %q claim", ClaimPasswordResetPassword)
	}

	cb, ok, err := tokenClaims.Claims.StringClaim(ClaimPasswordResetCallback)
	if err != nil {
		return PasswordReset{}, err
	}

	if _, err := url.Parse(cb); err != nil {
		return PasswordReset{}, fmt.Errorf("callback URL not parseable: %v", cb)
	}

	return PasswordReset{tokenClaims.Claims}, nil
}

func (e PasswordReset) UserID() string {
	return assertStringClaim(e.Claims, "sub")
}

func (e PasswordReset) Password() Password {
	pw := assertStringClaim(e.Claims, ClaimPasswordResetPassword)
	return Password(pw)
}

func (e PasswordReset) Callback() *url.URL {
	cb, ok, err := e.Claims.StringClaim(ClaimPasswordResetCallback)
	if err != nil {
		panic("PasswordReset: error getting string claim. This should be impossible.")
	}

	if !ok || cb == "" {
		return nil
	}

	cbURL, err := url.Parse(cb)
	if err != nil {
		panic("PasswordReset: can't parse callback. This should be impossible.")
	}
	return cbURL
}
