/*
Package translate implements translation of driver specific SQL queries.
*/
package translate

import (
	"database/sql"
	"regexp"

	"github.com/go-gorp/gorp"
)

var (
	bindRegexp = regexp.MustCompile(`\$\d+`)
	trueRegexp = regexp.MustCompile(`\btrue\b`)
)

// PostgresToSQLite translates github.com/lib/pq flavored SQL quries to github.com/mattn/go-sqlite3's flavor.
//
// It assumes that possitional bind arguements ($1, $2, etc.) are unqiue and in sequential order.
func PostgresToSQLite(query string) string {
	query = bindRegexp.ReplaceAllString(query, "?")
	query = trueRegexp.ReplaceAllString(query, "1")
	return query
}

// NewTranslatingExecutor returns an executor wrapping the existing executor. All query strings passed to
// the executor will be run through the translate function before begin passed to the driver.
func NewTranslatingExecutor(exec gorp.SqlExecutor, translate func(string) string) gorp.SqlExecutor {
	return &executor{exec, translate}
}

type executor struct {
	gorp.SqlExecutor
	Translate func(string) string
}

func (e *executor) Exec(query string, args ...interface{}) (sql.Result, error) {
	return e.SqlExecutor.Exec(e.Translate(query), args...)
}

func (e *executor) Select(i interface{}, query string, args ...interface{}) ([]interface{}, error) {
	return e.SqlExecutor.Select(i, e.Translate(query), args...)
}

func (e *executor) SelectInt(query string, args ...interface{}) (int64, error) {
	return e.SqlExecutor.SelectInt(e.Translate(query), args...)
}

func (e *executor) SelectNullInt(query string, args ...interface{}) (sql.NullInt64, error) {
	return e.SqlExecutor.SelectNullInt(e.Translate(query), args...)
}

func (e *executor) SelectFloat(query string, args ...interface{}) (float64, error) {
	return e.SqlExecutor.SelectFloat(e.Translate(query), args...)
}

func (e *executor) SelectNullFloat(query string, args ...interface{}) (sql.NullFloat64, error) {
	return e.SqlExecutor.SelectNullFloat(e.Translate(query), args...)
}

func (e *executor) SelectStr(query string, args ...interface{}) (string, error) {
	return e.SqlExecutor.SelectStr(e.Translate(query), args...)
}

func (e *executor) SelectNullStr(query string, args ...interface{}) (sql.NullString, error) {
	return e.SqlExecutor.SelectNullStr(e.Translate(query), args...)
}

func (e *executor) SelectOne(holder interface{}, query string, args ...interface{}) error {
	return e.SqlExecutor.SelectOne(holder, e.Translate(query), args...)
}
