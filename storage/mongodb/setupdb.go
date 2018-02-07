package mongodb

import (
	"strings"

	mgo "gopkg.in/mgo.v2"
)

// mongodb collection names.
const (
	ColAuthRequest     = "auth-request"
	ColClient          = "client"
	ColAuthCode        = "auth-code"
	ColRefreshToken    = "refresh-token"
	ColPassword        = "password"
	ColOfflineSessions = "offline-sessions"
	ColConnector       = "connector"

	// for storage keys
	ColSetting = "settings"
)

var mgoIndexs = map[string][]mgo.Index{
	ColAuthRequest: []mgo.Index{
		mgo.Index{
			Key:    []string{"id"},
			Unique: true,
		},
		mgo.Index{
			Key: []string{"clientid"},
		},
	},
	ColClient: []mgo.Index{
		mgo.Index{
			Key:    []string{"id"},
			Unique: true,
		},
	},
	ColAuthCode: []mgo.Index{
		mgo.Index{
			Key:    []string{"id"},
			Unique: true,
		},
	},
	ColRefreshToken: []mgo.Index{
		mgo.Index{
			Key:    []string{"id"},
			Unique: true,
		},
	},
	ColPassword: []mgo.Index{
		mgo.Index{
			Key:    []string{"email"},
			Unique: true,
		},
	},
	ColOfflineSessions: []mgo.Index{
		mgo.Index{
			Key:    []string{"userid", "connid"},
			Unique: true,
		},
	},
	ColConnector: []mgo.Index{
		mgo.Index{
			Key:    []string{"id"},
			Unique: true,
		},
	},
	ColSetting: []mgo.Index{
		mgo.Index{
			Key:    []string{"id"},
			Unique: true,
		},
	},
}

// setupIndex
func setupIndex(db *mgo.Database) error {
	for col, indexes := range mgoIndexs {
		for _, idx := range indexes {
			err := db.C(col).EnsureIndex(idx)
			if err != nil && !mgo.IsDup(err) {
				return err
			}
		}
	}
	return nil
}

func getDbNameFromURL(url string) string {
	url = strings.TrimPrefix(url, "mongodb://")

	i := strings.LastIndex(url, "/")
	if i <= 0 || i >= len(url)-1 {
		return defaultDBName
	}

	dbopt := string(url[i+1:])
	optI := strings.Index(dbopt, "?")
	if optI == 0 {
		return defaultDBName
	}
	if optI > 0 {
		return string(dbopt[:optI])
	}
	return dbopt
}
