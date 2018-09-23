package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/golang-migrate/migrate"
	"github.com/golang-migrate/migrate/database/postgres"
	_ "github.com/golang-migrate/migrate/source/file"
	_ "github.com/lib/pq"
)

type options struct {
	user, pass string
	host       string
	port       int
	dbName     string
	sslMode    string
}

func (o options) connectionInfo() string {
	return fmt.Sprintf("host='%s' port='%d' user='%s' password='%s' dbname='%s' sslmode='%s'",
		o.host, o.port, o.user, o.pass, o.dbName, o.sslMode)
}

func main() {
	var steps = flag.Int("steps", 0, "Number of steps to migrate. Leave unspecified to migrate all the way up or down.")
	var up = flag.Bool("up", false, "Migrate up. Cannot be used with -down.")
	var down = flag.Bool("down", false, "Migrate down. Cannot be used with -up.")
	var migrationstable = flag.String("migrationstable", "migrations", "Name of SQL table to store migrations in.")

	flag.Parse()

	// Neither are set or both are set
	if *up == *down {
		fmt.Printf("Error: must specify either -up or -down\n")
		return
	}

	port, err := strconv.Atoi(os.Getenv("PG_PORT"))
	if err != nil {
		port = 5432
		fmt.Printf("PG_PORT defaulted to: %d\n", port)
	}

	dbName, ok := os.LookupEnv("PG_DB_NAME")
	if !ok {
		dbName = "postgres"
		fmt.Printf("PG_DB_NAME defaulted to: %s\n", dbName)
	}

	o := options{
		user:    os.Getenv("PG_USER"),
		pass:    os.Getenv("PG_PASS"),
		host:    os.Getenv("PG_HOST"),
		port:    port,
		dbName:  dbName,
		sslMode: os.Getenv("PG_SSL_MODE"),
	}

	db, err := sql.Open("postgres", o.connectionInfo())
	if err != nil {
		fmt.Printf("Error: connecting to db: %v\n", err)
		return
	}
	defer db.Close()

	config := &postgres.Config{MigrationsTable: *migrationstable, DatabaseName: dbName}
	driver, err := postgres.WithInstance(db, config)
	if err != nil {
		fmt.Printf("Error: getting PostgreSQl driver: %v\n", err)
		return
	}
	m, err := migrate.NewWithDatabaseInstance(
		"file://./migrations/",
		dbName, driver)
	if err != nil {
		fmt.Printf("Error: creating migrations: %v\n", err)
		return
	}

	defer m.Close()

	if *steps == 0 && *up {
		err = m.Up()
	} else if *steps == 0 && *down {
		err = m.Down()
	} else {
		if *down {
			*steps = -*steps
		}
		err = m.Steps(*steps)
	}

	if err != nil {
		fmt.Printf("Error: running migrations: %v\n", err)
		return
	}

	fmt.Println("Migrations successfully run")

	srcErr, dbErr := m.Close()
	if srcErr != nil {
		fmt.Printf("Error: closing migrations source: %v\n", srcErr)
	}
	if dbErr != nil {
		fmt.Printf("Error: closing connection to database: %v\n", dbErr)
	}

	if err = db.Close(); err != nil {
		fmt.Printf("Error: closing database: %v\n", err)
	}
}
