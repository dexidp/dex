package user

import (
	"encoding/base64"
	"encoding/json"
)

// nextPageToken exists solely for JSON marshaling/unmarshaling nextPage params.
// It is not exported because we want nextPageTokens to be opaque and not rely
// on any specific encoding. However, because this encoding happens to be useful
// for both the in-mem and DB repo, we export the {Encode,Decode}NextPageToken
// functions.
type nextPageToken struct {
	Filter     UserFilter
	MaxResults int
	Offset     int
}

func EncodeNextPageToken(filter UserFilter, maxResults int, offset int) (string, error) {
	tok := nextPageToken{
		Filter:     filter,
		MaxResults: maxResults,
		Offset:     offset,
	}

	b, err := json.Marshal(&tok)
	if err != nil {
		return "", err
	}

	enc := base64.URLEncoding.EncodeToString(b)
	return enc, nil
}

func DecodeNextPageToken(tok string) (UserFilter, int, int, error) {
	b, err := base64.URLEncoding.DecodeString(tok)
	if err != nil {
		return UserFilter{}, 0, 0, err
	}

	var npt nextPageToken
	err = json.Unmarshal(b, &npt)
	if err != nil {
		return UserFilter{}, 0, 0, err
	}

	return npt.Filter, npt.MaxResults, npt.Offset, nil
}
