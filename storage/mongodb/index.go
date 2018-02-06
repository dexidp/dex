package mongodb

import (
	mgo "gopkg.in/mgo.v2"
)

const (
	ColAuthRequest     = "auth-request"
	ColClient          = "client"
	ColAuthCode        = "auth-code"
	ColRefreshToken    = "refresh-token"
	ColPassword        = "password"
	ColOfflineSessions = "offline-sessions"
	ColConnector       = "connector"
	ColSetting         = "settings" // for keys
)

var mgoIndexs = map[string][]mgo.Index{
	ColAuthRequest: []mgo.Index{
		mgo.Index{
			Key:    []string{"id"},
			Unique: true,
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

func setupIndex(db *mgo.Database) error {
	for col, indexes := range mgoIndexs {
		for _, idx := range indexes {
			db.C(col).EnsureIndex(idx)
		}
	}
	return nil
}
