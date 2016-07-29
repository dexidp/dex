package db

import (
	"errors"
	"fmt"

	"github.com/go-gorp/gorp"
	migrate "github.com/rubenv/sql-migrate"

	"github.com/coreos/dex/db/migrations"
)

const (
	migrationTable = "dex_migrations"
	migrationDir   = "db/migrations"
)

func init() {
	migrate.SetTable(migrationTable)
}

func MigrateToLatest(dbMap *gorp.DbMap) (int, error) {
	source, dialect, err := migrationSource(dbMap)
	if err != nil {
		return 0, err
	}
	return migrate.Exec(dbMap.Db, dialect, source, migrate.Up)
}

func MigrateMaxMigrations(dbMap *gorp.DbMap, max int) (int, error) {
	source, dialect, err := migrationSource(dbMap)
	if err != nil {
		return 0, err
	}
	return migrate.ExecMax(dbMap.Db, dialect, source, migrate.Up, max)
}

func GetPlannedMigrations(dbMap *gorp.DbMap) ([]*migrate.PlannedMigration, error) {
	source, dialect, err := migrationSource(dbMap)
	if err != nil {
		return nil, err
	}
	migrations, _, err := migrate.PlanMigration(dbMap.Db, dialect, source, migrate.Up, 0)
	return migrations, err
}

func DropMigrationsTable(dbMap *gorp.DbMap) error {
	qt := fmt.Sprintf("DROP TABLE IF EXISTS %s;", dbMap.Dialect.QuotedTableForQuery("", migrationTable))
	_, err := dbMap.Exec(qt)
	return err
}

func migrationSource(dbMap *gorp.DbMap) (src migrate.MigrationSource, dialect string, err error) {
	switch dbMap.Dialect.(type) {
	case gorp.PostgresDialect:
		return migrations.PostgresMigrations, "postgres", nil
	case gorp.SqliteDialect:
		src = &migrate.MemoryMigrationSource{
			Migrations: []*migrate.Migration{
				{
					Id: "dex.sql",
					Up: []string{sqlite3Migration},
				},
			},
		}

		return src, "sqlite3", nil
	default:
		return nil, "", errors.New("unsupported migration driver")
	}
}
