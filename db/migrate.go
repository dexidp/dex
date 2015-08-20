package db

import (
	"fmt"

	"github.com/coopernurse/gorp"
	"github.com/lib/pq"
	migrate "github.com/rubenv/sql-migrate"

	"github.com/coreos/dex/db/migrations"
)

const (
	migrationDialect = "postgres"
	migrationTable   = "dex_migrations"
	migrationDir     = "db/migrations"
)

func init() {
	migrate.SetTable(migrationTable)
}

func MigrateToLatest(dbMap *gorp.DbMap) (int, error) {
	source := getSource()

	return migrate.Exec(dbMap.Db, migrationDialect, source, migrate.Up)
}

func MigrateMaxMigrations(dbMap *gorp.DbMap, max int) (int, error) {
	source := getSource()

	return migrate.ExecMax(dbMap.Db, migrationDialect, source, migrate.Up, max)
}

func GetPlannedMigrations(dbMap *gorp.DbMap) ([]*migrate.PlannedMigration, error) {
	migrations, _, err := migrate.PlanMigration(dbMap.Db, migrationDialect, getSource(), migrate.Up, 0)
	return migrations, err
}

func DropMigrationsTable(dbMap *gorp.DbMap) error {
	qt := pq.QuoteIdentifier(migrationTable)
	_, err := dbMap.Exec(fmt.Sprintf("drop table if exists %s ;", qt))
	return err
}

func getSource() migrate.MigrationSource {
	return &migrate.AssetMigrationSource{
		Dir:      migrationDir,
		Asset:    migrations.Asset,
		AssetDir: migrations.AssetDir,
	}
}
