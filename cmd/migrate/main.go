package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Pigmice2733/peregrine-backend/internal/config"
	"github.com/golang-migrate/migrate"
	"github.com/golang-migrate/migrate/database/postgres"
	_ "github.com/golang-migrate/migrate/source/file"
	_ "github.com/lib/pq"
	"github.com/pkg/errors"
)

func main() {
	var steps = flag.Int("steps", 0, "Number of steps to migrate. Leave unspecified to migrate all the way up or down.")
	var up = flag.Bool("up", false, "Migrate up. Cannot be used with -down.")
	var down = flag.Bool("down", false, "Migrate down. Cannot be used with -up.")
	var migrationsTable = flag.String("migrationsTable", "migrations", "Name of SQL table to store migrations in.")
	var basePath = flag.String("basePath", "", "Path to the etc directory where the config file is.")

	flag.Parse()

	if err := run(*steps, *up, *down, *migrationsTable, *basePath); err != nil {
		fmt.Printf("got error: %v\n", err)
		os.Exit(1)
	}
}

func run(steps int, up, down bool, migrationsTable, basePath string) error {
	// Neither are set or both are set
	if up == down {
		return fmt.Errorf("must specify either -up or -down")
	}

	c, err := config.Open(basePath)
	if err != nil {
		return errors.Wrap(err, "opening config")
	}

	db, err := sql.Open("postgres", c.Database.ConnectionInfo())
	if err != nil {
		return errors.Wrap(err, "connecting to database")
	}
	defer db.Close()

	config := &postgres.Config{MigrationsTable: migrationsTable, DatabaseName: c.Database.Name}
	driver, err := postgres.WithInstance(db, config)
	if err != nil {
		return errors.Wrap(err, "getting postgresql driver")
	}

	migrationsSource := "file://"
	if basePath != "" {
		migrationsSource += filepath.ToSlash(filepath.Clean(basePath)) + "/"
	} else {
		migrationsSource += "./"
	}
	migrationsSource += "migrations"

	m, err := migrate.NewWithDatabaseInstance(
		migrationsSource,
		c.Database.Name, driver)
	if err != nil {
		return errors.Wrap(err, "opening migrations")
	}

	defer m.Close()

	if steps == 0 && up {
		err = m.Up()
	} else if steps == 0 && down {
		err = m.Down()
	} else {
		if down {
			steps = -steps
		}
		err = m.Steps(steps)
	}

	return errors.Wrap(err, "running migrations")
}
