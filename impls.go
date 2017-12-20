package main

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	driver "github.com/arangodb/go-driver"
	"github.com/arangodb/go-driver/http"
)

const (
	migCol string = "arangomigo"
)

// Migration all the operations necessary to modify a database, even make one.
type Migration interface {
	migrate(ctx context.Context, driver *driver.Database) error
	FileName() string
	SetFileName(name string)
	CheckSum() string
	SetCheckSum(sum string)
}

// FileName gets the filename of the migrations configuration.
func (op *Operation) FileName() string {
	return op.fileName
}

// SetFileName updates the filename of the migration
func (op *Operation) SetFileName(fileName string) {
	op.fileName = fileName
}

// CheckSum gets the checksum for the migration's file
func (op *Operation) CheckSum() string {
	return op.checksum
}

// SetCheckSum sets the checksum of the file, in hex.
func (op *Operation) SetCheckSum(sum string) {
	op.checksum = sum
}

// End Common operation implementations

// Entry point in actually executing the migrations
func perform(ctx context.Context, c Config) error {
	pm, err := migrations(c.MigrationsPath)
	if e(err) {
		return err
	}

	cl, err := client(ctx, c)
	db, err := loadDb(ctx, c, cl, &pm)
	if e(err) {
		return err
	}
	err = migrateNow(ctx, db, pm)
	return err
}

// Processed marker. Declared here since it's impl related.
type migration struct {
	Key      string `json:"_key"`
	Checksum string
}

func migrateNow(ctx context.Context, db driver.Database, pms []PairedMigrations) error {
	fmt.Println("Starting migration now")

	mcol, err := db.Collection(ctx, migCol)
	if e(err) {
		return err
	}

	for _, pm := range pms {
		m := pm.change
		u := pm.undo

		// Since migrations are stored by their file names, just see if it exists
		migRan, err := mcol.DocumentExists(ctx, m.FileName())
		if e(err) {
			return err
		}

		if !migRan {
			err := m.migrate(ctx, &db)
			if !e(err) {
				if temp, ok := m.(*Database); !ok || temp.Action == MODIFY {
					_, err := mcol.CreateDocument(ctx, &migration{Key: m.FileName(), Checksum: m.CheckSum()})
					if e(err) {
						return err
					}
				}
			} else if e(err) && driver.IsArangoError(err) && u != nil {
				// This probably means a migration issue, back out.
				err = u.migrate(ctx, &db)
				if e(err) {
					return err
				}
			} else {
				return err
			}
		}
	}
	return nil
}

func loadDb(
	ctx context.Context,
	conf Config,
	cl driver.Client,
	pm *[]PairedMigrations) (driver.Database, error) {
	// Checks to see if the database exists
	dbName := conf.Db
	db, err := cl.Database(ctx, dbName)
	if err != nil && driver.IsNotFound(err) {
		// Creating a database requires extra setup.
		m := (*pm)[0].change
		o, ok := m.(*Database)
		if !ok {
			return nil, errors.Errorf("Database %s does not exist and first migration is not the DB creation", dbName)
		}
		if o.Name != dbName {
			return nil, errors.New("Configuration's dbname does not match migration name")
		}
		o.cl = cl
		err = m.migrate(ctx, &db)
		if err == nil {
			db = o.db
			fmt.Printf("Target db is now %s\n", db.Name())
		}
	} else if err == nil {
		m := (*pm)[0].change
		switch m.(type) {
		case *Database:
			*pm = (*pm)[1:]
		}
	}

	if err == nil {
		// Check to see if the migration coll is there.
		_, err := db.Collection(ctx, migCol)
		if driver.IsNotFound(err) {
			ko := driver.CollectionKeyOptions{}
			ko.AllowUserKeys = true
			options := driver.CreateCollectionOptions{}
			options.KeyOptions = &ko
			db.CreateCollection(ctx, migCol, &options)
		}
	}

	return db, err
}

// Create the client used to talk to ArangoDB
func client(ctx context.Context, c Config) (driver.Client, error) {
	conn, err := http.NewConnection(http.ConnectionConfig{
		Endpoints: c.Endpoints,
	})

	if e(err) {
		return nil, errors.New("Couldn't create connection to Arango\n" + err.Error())
	}
	cl, err := driver.NewClient(driver.ClientConfig{
		Connection:     conn,
		Authentication: driver.BasicAuthentication(c.Username, c.Password),
	})

	return cl, err
}

func e(err error) bool {
	return err != nil
}

func (d *Database) migrate(ctx context.Context, db *driver.Database) error {
	var oerr error
	switch d.Action {
	case CREATE:
		if d.db != nil { // no idea why this works.
			return nil
		}
		options := driver.CreateDatabaseOptions{}
		active := true
		for _, u := range d.Allowed {
			options.Users = append(options.Users,
				driver.CreateDatabaseUserOptions{
					UserName: u.Username,
					Password: u.Password,
					Active:   &active,
				})
		}
		newdb, err := d.cl.CreateDatabase(ctx, d.Name, &options)
		if err == nil {
			d.db = newdb
		} else {
			oerr = err
		}
	case DELETE:
		err := (*db).Remove(ctx)
		if e(err) {
			oerr = err
		}
	default:
		oerr = errors.Errorf("Database migration does not support op %s", d.Action)
	}

	return oerr
}

func (cl Collection) migrate(ctx context.Context, db *driver.Database) error {
	d := *db
	switch cl.Action {
	case CREATE:
		options := driver.CreateCollectionOptions{}
		options.DoCompact = &cl.Compactable
		options.JournalSize = cl.JournalSize
		options.WaitForSync = cl.WaitForSync
		options.ShardKeys = cl.ShardKeys
		options.IsVolatile = cl.Volatile

		// Configures the user keys
		ko := driver.CollectionKeyOptions{}
		ko.AllowUserKeys = cl.AllowUserKeys
		options.KeyOptions = &ko

		_, err := d.CreateCollection(ctx, cl.Name, &options)
		if e(err) {
			return err
		}
	}
	return nil
}

func (g Graph) migrate(ctx context.Context, db *driver.Database) error {
	d := *db

	switch g.Action {
	case CREATE:
		options := driver.CreateGraphOptions{}
		options.IsSmart = g.Smart
		options.SmartGraphAttribute = g.SmartGraphAttribute

		numShards := 1
		if g.Shards > 0 {
			numShards = g.Shards
		}

		options.NumberOfShards = numShards

		for _, ed := range g.EdgeDefinitions {
			options.EdgeDefinitions = append(
				options.EdgeDefinitions,
				driver.EdgeDefinition{
					Collection: ed.Collection,
					To:         ed.To,
					From:       ed.From,
				})
		}

		options.OrphanVertexCollections = g.OrphanVertex

		_, err := d.CreateGraph(ctx, g.Name, &options)
		return err
	case DELETE:
		aG, err := d.Graph(ctx, g.Name)
		if e(err) {
			return errors.Wrapf(err, "Couldn't find graph with name %s. Can't delete.", g.Name)
		}
		return errors.Wrapf(aG.Remove(ctx), "Couldn't remove graph %s", g.Name)
	}
	return nil
}
