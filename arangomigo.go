/*
Package main allows the tool to execute from the command line.
*/
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/solher/arangolite"
	"github.com/solher/arangolite/requests"
	"log"
	"strings"
	"time"
)

const (
	WAIT_ON_NO_DB = 0
	DEFAULT_DB    = "_system"
)

func main() {
	config := load()
	if err := migrate(config); err != nil {
		log.Fatal("Could not perform migration\n", err)
	}
	fmt.Println("Successfully completed migration")
}

// TODO remember that having replayable migrations need to be possible too.
// Have branch those into running at the end.

func migrate(c Config) error {
	ctx := context.Background()

	dbName := c.db
	db := arangolite.NewDatabase(
		arangolite.OptEndpoint(c.host),
		arangolite.OptBasicAuth(c.user, c.password),
		arangolite.OptDatabaseName(dbName),
	)

	if err := db.Connect(ctx); err != nil {
		// Assume that this is a new database
		if strings.Contains(err.Error(), "database not found") {
			fmt.Printf("Database %s does not exist. Assuming this is a new install.\nWaiting %d seconds to allow cancel.\n", c.db, WAIT_ON_NO_DB)
			time.Sleep(time.Duration(WAIT_ON_NO_DB) * time.Second)
			dbName = DEFAULT_DB
			db.Options(arangolite.OptDatabaseName(dbName))
		} else {
			return err
		}
	}

	if err := perform(ctx, db, c.migrationsPath, dbName); err != nil {
		return err
	}
	return nil
}

func perform(
	ctx context.Context, db *arangolite.Database,
	path string, currentDB string,
) error {
	ms, err := migrations(path)

	if err != nil {
		return err
	}

	ms, err = confirmValid(ctx, db, ms, currentDB)
	if err != nil {
		return err
	}

	return nil
}

// Gets the ball rolling on migration.
func confirmValid(ctx context.Context, db *arangolite.Database, ms []PairedMigrations, curDb string) ([]PairedMigrations, error) {

	isDBM := func(t interface{}) bool {
		switch t.(type) {
		case *Database:
			return true
		default:
			return false
		}
	}

	m := ms[0].change
	if curDb == DEFAULT_DB && !isDBM(m) {
		return nil, errors.New("The database does not exists and the first migration is not creating a database.")
	}

	// Create the database, and switch to it.
	if curDb == DEFAULT_DB && isDBM(m) {
		o, _ := m.(*Database)
		if o.Action != CREATE {
			return nil, errors.New("First migration is not creating a database")
		}
		if err := m.migrate(ctx, db); err != nil {
			return nil, err
		}

		// adjust db context
		dbName := o.Name
		db.Options(
			arangolite.OptDatabaseName(dbName),
		)
		userName := ""
		if len(o.Allowed) > 0 {
			user := o.Allowed[0]
			userName = user.Username
			db.Options(arangolite.OptBasicAuth(userName, user.Password))
		}
		fmt.Printf("Switched to database '%s' with user '%s'\n", dbName, userName)
		ms = ms[1:]
	}

	// Add migration collection
	cols := requests.CollectionInfoList{}
	err := db.Run(ctx, &cols, &requests.ListCollections{})
	fmt.Printf("Cols are %v", cols)
	return ms, err
}

func load() Config {
	host := flag.String("host", "http://localhost:8529", "The url for the host with protocol, host and port")
	user := flag.String("user", "", "the username for root like arangodb powers")
	password := flag.String("password", "", "The password for the account with root like powers")
	useSSL := flag.Bool("useSSL", false, "Existance indicates that the connection should use TLS")
	migrationPath := flag.String("migrationsPath", "", "Location of the directory holding the migrations")
	action := flag.String("action", "up", "Indicates which migration action to use: up or down")
	db := flag.String("db", "", "The database you want to connect with")
	flag.Parse()

	return Config{
		host:           *host,
		user:           *user,
		password:       *password,
		ssl:            *useSSL,
		migrationsPath: *migrationPath,
		action:         *action,
		db:             *db,
	}
}

type Config struct {
	host           string
	user           string
	password       string
	ssl            bool
	migrationsPath string
	action         string
	db             string
}
