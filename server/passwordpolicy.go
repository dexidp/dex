package server

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/dexidp/dex/storage"
)

type PasswordPolicyOption func(*PasswordPolicy) error

type complexityLevel int

const (
	levelNone complexityLevel = iota
	levelLow
	levelFair
	levelGood
	levelExcellent
)

func (cl complexityLevel) String() string {
	switch cl {
	case levelNone:
		return "none"
	case levelLow:
		return "low"
	case levelFair:
		return "fair"
	case levelGood:
		return "good"
	case levelExcellent:
		return "excellent"
	default:
		return "unknown"
	}
}

type Complexity struct {
	level      complexityLevel
	userPrompt string
}

func (cl Complexity) UserPrompt() string {
	return cl.userPrompt
}

var (
	ComplexityNone = Complexity{levelNone, ""}
	ComplexityLow  = Complexity{levelLow, "Password must contain:\n" +
		"• At least 8 characters"}
	ComplexityFair = Complexity{levelFair, "Password must contain:\n" +
		"• At least 8 characters\n" +
		"• Upper and lowercase letters\n" +
		"• At least one number"}
	ComplexityGood = Complexity{levelGood, "Password must contain:\n" +
		"• At least 8 characters\n" +
		"• Upper and lowercase letters\n" +
		"• At least one number\n" +
		"• Special characters (!@#$%^&* etc.)"}
	ComplexityExcellent = Complexity{levelExcellent, "Password must contain:\n" +
		"• At least 8 characters\n" +
		"• Upper and lowercase letters\n" +
		"• At least one number\n" +
		"• Special characters (!@#$%^&* etc.)\n" +
		"• No more than 2 identical characters in a row"}
)

func (cl Complexity) Validate(password string) error {
	switch cl.level {
	case levelNone:
		return nil

	case levelLow:
		if len(password) < 8 {
			return errors.New("minimum 8 characters required")
		}
		return nil

	case levelFair:
		if len(password) < 8 {
			return errors.New("minimum 8 characters required")
		}
		var hasLower, hasUpper, hasNumber bool
		for _, c := range password {
			switch {
			case unicode.IsLower(c):
				hasLower = true
			case unicode.IsUpper(c):
				hasUpper = true
			case unicode.IsNumber(c):
				hasNumber = true
			}
		}
		if !hasLower {
			return errors.New("at least one lowercase letter required")
		}
		if !hasUpper {
			return errors.New("at least one uppercase letter required")
		}
		if !hasNumber {
			return errors.New("at least one number required")
		}
		return nil
	case levelGood:
		if len(password) < 8 {
			return errors.New("minimum 8 characters required")
		}
		var hasLower, hasUpper, hasNumber, hasSpecial bool
		for _, c := range password {
			switch {
			case unicode.IsLower(c):
				hasLower = true
			case unicode.IsUpper(c):
				hasUpper = true
			case unicode.IsNumber(c):
				hasNumber = true
			case !unicode.IsLetter(c) && !unicode.IsNumber(c):
				hasSpecial = true
			}
		}
		if !hasLower {
			return errors.New("at least one lowercase letter required")
		}
		if !hasUpper {
			return errors.New("at least one uppercase letter required")
		}
		if !hasNumber {
			return errors.New("at least one number required")
		}
		if !hasSpecial {
			return errors.New("at least one special character (!@#$ etc.) required")
		}
		return nil
	case levelExcellent:
		if len(password) < 8 {
			return errors.New("minimum 8 characters required")
		}
		var hasLower, hasUpper, hasNumber, hasSpecial bool
		var previousLetter rune
		for _, c := range password {
			if c == previousLetter {
				return errors.New("password contains 2 identical characters in a row")
			}
			switch {
			case unicode.IsLower(c):
				hasLower = true
			case unicode.IsUpper(c):
				hasUpper = true
			case unicode.IsNumber(c):
				hasNumber = true
			case !unicode.IsLetter(c) && !unicode.IsNumber(c):
				hasSpecial = true
			}
			previousLetter = c
		}
		if !hasLower {
			return errors.New("at least one lowercase letter required")
		}
		if !hasUpper {
			return errors.New("at least one uppercase letter required")
		}
		if !hasNumber {
			return errors.New("at least one number required")
		}
		if !hasSpecial {
			return errors.New("at least one special character (!@#$ etc.) required")
		}
		return nil
	default:
		return errors.New("unknown password policy level")
	}
}

func GetComplexity(level string) (Complexity, error) {
	switch strings.ToLower(level) {
	case levelNone.String(), "":
		return ComplexityNone, nil
	case levelLow.String():
		return ComplexityLow, nil
	case levelFair.String():
		return ComplexityFair, nil
	case levelGood.String():
		return ComplexityGood, nil
	case levelExcellent.String():
		return ComplexityExcellent, nil
	default:
		return Complexity{}, fmt.Errorf("unknown password complexity level: %s", level)
	}
}

func WithComplexityLevel(complexityLevel string) PasswordPolicyOption {
	return func(pp *PasswordPolicy) error {
		cl, err := GetComplexity(complexityLevel)
		if err != nil {
			return err
		}
		pp.complexity = cl
		return nil
	}
}

type LockoutSettings struct {
	maxAttempts  uint64
	lockDuration time.Duration
}

func WithLockoutSettings(maxAttempts uint64, lockDuration time.Duration) PasswordPolicyOption {
	return func(pp *PasswordPolicy) error {
		pp.lockout = &LockoutSettings{
			maxAttempts:  maxAttempts,
			lockDuration: lockDuration,
		}
		return nil
	}
}

type RotationSettings struct {
	passwordRotationInterval time.Duration
	redirectURL              *string
}

func WithRotationSettings(rotationInterval time.Duration, redirectURL *string) PasswordPolicyOption {
	return func(pp *PasswordPolicy) error {
		pp.rotation = &RotationSettings{
			passwordRotationInterval: rotationInterval,
			redirectURL:              redirectURL,
		}
		return nil
	}
}

func WithPasswordHistoryLimit(passwordHistoryLimit uint64) PasswordPolicyOption {
	return func(pp *PasswordPolicy) error {
		pp.passwordHistoryLimit = passwordHistoryLimit
		return nil
	}
}

type PasswordPolicy struct {
	id string
	// complexityLevel Sets password complexity requirements
	// Possible values:
	//   - None:      No restrictions. Password can be any length starting from 1 character.
	//   - Low:       Minimum 8 characters.
	//   - Fair:      (Default) Minimum 8 characters, at least:
	//                • One uppercase letter
	//                • One lowercase letter
	//                • One digit
	//   - Good:      Minimum 8 characters, at least:
	//                • One uppercase letter
	//                • One lowercase letter
	//                • One digit
	//                • One special character (!@#$%^&* etc.)
	//   - Excellent: Minimum 8 characters, at least:
	//                • One uppercase letter
	//                • One lowercase letter
	//                • One digit
	//                • One special character (!@#$%^&* etc.)
	//                • No more than 2 identical characters in a row
	complexity Complexity
	// passwordHistoryLimit Sets count of stored passwords.
	// Prevents user from reuse old passwords.
	passwordHistoryLimit uint64
	// lockout Settings that define restrictions and conditions for account lockout after failed login attempts.
	lockout *LockoutSettings
	// rotation Settings defines rule for periodic credential rotation.
	rotation *RotationSettings
}

func (pp PasswordPolicy) IsPasswordLocked(p storage.Password) bool {
	if pp.lockout != nil && p.LockedUntil != nil && p.LockedUntil.After(time.Now()) {
		return true
	}
	return false
}

func (pp PasswordPolicy) IsMaxLoginAttemptsExeeded(attempts uint64) bool {
	if pp.lockout != nil && pp.lockout.maxAttempts <= attempts {
		return true
	}
	return false
}

func (pp PasswordPolicy) IsPasswordExpired(passwordCreatedAt time.Time) bool {
	if pp.rotation == nil {
		return false
	}

	passwordExpirationDate := passwordCreatedAt.Add(pp.rotation.passwordRotationInterval)

	return passwordExpirationDate.Before(time.Now())
}

func NewPasswordPolicy(id string, options ...PasswordPolicyOption) (*PasswordPolicy, error) {
	if id == "" {
		return nil, fmt.Errorf("password policy id cannot be empty")
	}
	pp := &PasswordPolicy{
		id:         id,
		complexity: ComplexityFair,
	}

	for _, opt := range options {
		if err := opt(pp); err != nil {
			return nil, err
		}
	}

	return pp, nil
}
