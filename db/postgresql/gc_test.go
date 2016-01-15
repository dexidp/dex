package db

import (
	"errors"
	"reflect"
	"testing"
)

type staticPurger struct {
	err error
}

func (p staticPurger) purge() error {
	return p.err
}

func TestPurgeAll(t *testing.T) {
	tests := []struct {
		pm   []namedPurger
		want []purgeError
	}{
		{
			pm: []namedPurger{
				namedPurger{
					name:   "foo",
					purger: staticPurger{err: nil},
				},
			},
			want: []purgeError{},
		},
		{
			pm: []namedPurger{
				namedPurger{
					name:   "foo",
					purger: staticPurger{err: errors.New("foo fail")},
				},
			},
			want: []purgeError{
				purgeError{name: "foo", err: errors.New("foo fail")},
			},
		},

		{
			pm: []namedPurger{
				namedPurger{
					name:   "foo",
					purger: staticPurger{err: nil},
				},
				namedPurger{
					name:   "bar",
					purger: staticPurger{err: errors.New("bar fail")},
				},
				namedPurger{
					name:   "baz",
					purger: staticPurger{err: nil},
				},
				namedPurger{
					name:   "fum",
					purger: staticPurger{err: errors.New("fum fail")},
				},
			},
			want: []purgeError{
				purgeError{name: "bar", err: errors.New("bar fail")},
				purgeError{name: "fum", err: errors.New("fum fail")},
			},
		},
	}

	for i, tt := range tests {
		got := make([]purgeError, 0)
		for perr := range purgeAll(tt.pm) {
			got = append(got, perr)
		}
		if !reflect.DeepEqual(tt.want, got) {
			t.Errorf("case %d: want=%v, got=%v", i, tt.want, got)
		}
	}
}

func TestAnyPurgeErrors(t *testing.T) {
	tests := []struct {
		chanFunc func() chan purgeError
		want     bool
	}{
		{
			chanFunc: func() chan purgeError {
				errchan := make(chan purgeError)
				close(errchan)
				return errchan
			},
			want: false,
		},

		{
			chanFunc: func() chan purgeError {
				errchan := make(chan purgeError, 1)
				errchan <- purgeError{}
				close(errchan)
				return errchan
			},
			want: true,
		},

		{
			chanFunc: func() chan purgeError {
				errchan := make(chan purgeError, 4)
				errchan <- purgeError{}
				errchan <- purgeError{}
				errchan <- purgeError{}
				errchan <- purgeError{}
				close(errchan)
				return errchan
			},
			want: true,
		},
	}

	for i, tt := range tests {
		errchan := tt.chanFunc()
		got := anyPurgeErrors(errchan)
		if tt.want != got {
			t.Errorf("case %d: want=%t got=%t", i, tt.want, got)
		}
	}
}
