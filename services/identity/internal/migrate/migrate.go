package migrate

import (
	"embed"
	"fmt"

	migrate_lib "github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed *.sql
var migrationsFS embed.FS

// Down rolls back the most recent migration.
func Down(dbURL string) error {
	d, err := iofs.New(migrationsFS, ".")
	if err != nil {
		return fmt.Errorf("migrate down: open source: %w", err)
	}
	m, err := migrate_lib.NewWithSourceInstance("iofs", d, dbURL)
	if err != nil {
		return fmt.Errorf("migrate down: new instance: %w", err)
	}
	defer m.Close()

	if err := m.Steps(-1); err != nil && err != migrate_lib.ErrNoChange {
		return fmt.Errorf("migrate down: %w", err)
	}
	return nil
}

func Run(dbURL string) error {
	d, err := iofs.New(migrationsFS, ".")
	if err != nil {
		return fmt.Errorf("open embedded migrations: %w", err)
	}
	m, err := migrate_lib.NewWithSourceInstance("iofs", d, dbURL)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate_lib.ErrNoChange {
		return fmt.Errorf("run migrations: %w", err)
	}
	return nil
}
