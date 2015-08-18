package db

import (
	"time"

	"github.com/coopernurse/gorp"
	"github.com/jonboulle/clockwork"

	"github.com/coreos/dex/pkg/log"
	ptime "github.com/coreos/dex/pkg/time"
)

type purger interface {
	purge() error
}

type namedPurger struct {
	name string
	purger
}

func NewGarbageCollector(dbm *gorp.DbMap, ival time.Duration) *GarbageCollector {
	sRepo := NewSessionRepo(dbm)
	skRepo := NewSessionKeyRepo(dbm)

	purgers := []namedPurger{
		namedPurger{
			name:   "session",
			purger: sRepo,
		},
		namedPurger{
			name:   "session_key",
			purger: skRepo,
		},
	}

	gc := GarbageCollector{
		purgers:  purgers,
		interval: ival,
		clock:    clockwork.NewRealClock(),
	}

	return &gc
}

type GarbageCollector struct {
	purgers  []namedPurger
	interval time.Duration
	clock    clockwork.Clock
}

func (gc *GarbageCollector) Run() chan struct{} {
	stop := make(chan struct{})

	go func() {
		var failing bool
		next := gc.interval
		for {
			select {
			case <-gc.clock.After(next):
				if anyPurgeErrors(purgeAll(gc.purgers)) {
					if !failing {
						failing = true
						next = time.Second
					} else {
						next = ptime.ExpBackoff(next, time.Minute)
					}
					log.Errorf("Failed garbage collection, retrying in %v", next)
				} else {
					failing = false
					next = gc.interval
					log.Infof("Garbage collection complete, running again in %v", next)
				}
			case <-stop:
				return
			}
		}
	}()

	return stop
}

type purgeError struct {
	name string
	err  error
}

func anyPurgeErrors(errchan <-chan purgeError) (found bool) {
	for perr := range errchan {
		found = true
		log.Errorf("Failed purging %s: %v", perr.name, perr.err)
	}
	return
}

func purgeAll(purgers []namedPurger) <-chan purgeError {
	errchan := make(chan purgeError)
	go func() {
		for _, p := range purgers {
			if err := p.purge(); err != nil {
				errchan <- purgeError{name: p.name, err: err}
			}
		}
		close(errchan)
	}()
	return errchan
}
