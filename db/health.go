package db

import (
	"errors"
	"fmt"

	"github.com/go-gorp/gorp"
)

func NewHealthChecker(dbm *gorp.DbMap) *healthChecker {
	return &healthChecker{dbMap: dbm}
}

type healthChecker struct {
	dbMap *gorp.DbMap
}

func (hc *healthChecker) Healthy() (err error) {
	if err = hc.dbMap.Db.Ping(); err != nil {
		err = fmt.Errorf("database error: %v", err)
		return
	}

	num, err := hc.dbMap.SelectInt("SELECT 1")
	if err != nil {
		return
	}

	if num != 1 {
		err = errors.New("unable to connect to database")
	}

	return
}
