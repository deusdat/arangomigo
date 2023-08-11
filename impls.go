package arangomigo

import (
	"context"
	"crypto/md5"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"log"

	"github.com/pkg/errors"

	"github.com/arangodb/go-driver"
	"github.com/arangodb/go-driver/http"
)

const (
	migCol string = "arangomigo"
)

// Migration all the operations necessary to modify a database, even make one.
type Migration interface {
	Migrate(ctx context.Context, driver driver.Database, extras map[string]interface{}) error
	FileName() string
	SetFileName(name string)
	CheckSum() string
	SetCheckSum(sum string)
}

// FileName gets the filename of the Migrations configuration.
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

func PerformMigrations(ctx context.Context, c Config, ms []Migration) error {
	var pms []PairedMigrations
	for i, migration := range ms {
		name := fmt.Sprintf("%d.migration", i)
		migration.SetFileName(name)
		chk := md5.Sum([]byte(name))
		migration.SetCheckSum(hex.EncodeToString(chk[:]))
		pms = append(pms, PairedMigrations{change: migration, undo: nil})
	}
	return Perform(ctx, c, pms)
}

// Perform is the entry point in actually executing the Migrations
func Perform(ctx context.Context, c Config, pm []PairedMigrations) error {
	cl, err := client(c)
	db, err := loadDb(ctx, c, cl, &pm, c.Extras)
	if e(err) {
		return err
	}
	err = migrateNow(ctx, db, pm, c.Extras)
	return err
}

// Processed marker. Declared here since it's impl related.
type migration struct {
	Key      string `json:"_key"`
	Checksum string
}

func migrateNow(
	ctx context.Context,
	db driver.Database,
	pms []PairedMigrations,
	extras map[string]interface{},
) error {
	log.Println("Starting migration now")

	mcol, err := db.Collection(ctx, migCol)
	if e(err) {
		return err
	}

	for _, pm := range pms {
		m := pm.change
		u := pm.undo

		// Since Migrations are stored by their file names, just see if it exists
		migRan, err := mcol.DocumentExists(ctx, m.FileName())
		if e(err) {
			return err
		}

		if !migRan {
			err := m.Migrate(ctx, db, extras)
			if !e(err) {
				if temp, ok := m.(*Database); !ok || temp.Action == MODIFY {
					_, err := mcol.CreateDocument(ctx, &migration{Key: m.FileName(), Checksum: m.CheckSum()})
					if e(err) {
						return err
					}
				}
			} else if e(err) && driver.IsArangoError(err) && u != nil {
				// This probably means a migration issue, back out.
				err = u.Migrate(ctx, db, extras)
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

func pointyBool(bool2 bool) *bool {
	return &bool2
}

func loadDb(
	ctx context.Context,
	conf Config,
	cl driver.Client,
	pm *[]PairedMigrations,
	extras map[string]interface{}) (driver.Database, error) {
	// Checks to see if the database exists
	dbName := conf.Db
	db, err := cl.Database(ctx, dbName)
	if err != nil && driver.IsNotFoundGeneral(err) {
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
		err = m.Migrate(ctx, db, extras)
		if err == nil {
			db = o.db
			log.Printf("Target db is now %s\n", db.Name())
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
		if driver.IsNotFoundGeneral(err) {
			ko := driver.CollectionKeyOptions{}
			ko.AllowUserKeysPtr = pointyBool(true)
			options := driver.CreateCollectionOptions{}
			options.KeyOptions = &ko
			if _, err := db.CreateCollection(ctx, migCol, &options); err != nil {
				log.Printf("Failed to create collection %s", migCol)
				return db, err
			}
		}
	}

	return db, err
}

// Create the client used to talk to ArangoDB
func client(c Config) (driver.Client, error) {
	conn, err := http.NewConnection(http.ConnectionConfig{
		Endpoints: c.Endpoints,
		TLSConfig: &tls.Config{
			InsecureSkipVerify: c.SkipSslVerify,
		},
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

func (d *Database) Migrate(ctx context.Context, db driver.Database, extras map[string]interface{}) error {
	var oerr error
	switch d.Action {
	case CREATE:
		if d.db != nil { // no idea why this works.
			return nil
		}
		options := driver.CreateDatabaseOptions{}
		active := true
		for _, u := range d.Allowed {
			options.Users = append(
				options.Users,
				driver.CreateDatabaseUserOptions{
					UserName: directReplace(u.Username, extras).(string),
					Password: directReplace(u.Password, extras).(string),
					Active:   &active,
				},
			)
		}
		newdb, err := d.cl.CreateDatabase(ctx, d.Name, &options)
		if err == nil {
			d.db = newdb
		} else {
			oerr = err
		}
	case DELETE:
		err := db.Remove(ctx)
		if e(err) {
			oerr = err
		}
	default:
		oerr = errors.Errorf("Database migration does not support op %s", d.Action)
	}

	return errors.Wrap(oerr, "Couldn't create database")
}

// directReplace attempts to use the key value to find a lookup in the map.
// if one exists, it returns the values; otherwise returns the key.
func directReplace(key string, extras map[string]interface{}) interface{} {
	if val, ok := extras[key]; ok {
		return val
	}
	return key
}

func (cl Collection) Migrate(ctx context.Context, db driver.Database, _ map[string]interface{}) error {
	switch cl.Action {
	case CREATE:
		options := driver.CreateCollectionOptions{}
		if cl.Compactable != nil {
			options.DoCompact = cl.Compactable
		}
		if cl.JournalSize != nil {
			options.JournalSize = *cl.JournalSize
		}
		if cl.WaitForSync != nil {
			options.WaitForSync = *cl.WaitForSync
		}
		if cl.ShardKeys != nil {
			options.ShardKeys = *cl.ShardKeys
		}
		if cl.Volatile != nil {
			options.IsVolatile = *cl.Volatile
		}
		if cl.CollectionType != "" {
			options.Type = driver.CollectionTypeDocument
			if cl.CollectionType == "edge" {
				options.Type = driver.CollectionTypeEdge
			}
		}
		// Configures the user keys
		ko := driver.CollectionKeyOptions{}
		if cl.AllowUserKeys != nil {
			ko.AllowUserKeysPtr = cl.AllowUserKeys
		}
		options.KeyOptions = &ko

		_, err := db.CreateCollection(ctx, cl.Name, &options)
		if e(err) {
			return err
		}
	case DELETE:
		col, err := db.Collection(ctx, cl.Name)
		if e(err) {
			return errors.Wrapf(err, "Couldn't find collection '%s' to delete", cl.Name)
		}
		err = col.Remove(ctx)
		if !e(err) {
			log.Printf("Deleted collection '%s'\n", cl.Name)
		}
		return errors.Wrapf(err, "Couldn't delete collection '%s'.", cl.Name)
	case MODIFY:
		col, err := db.Collection(ctx, cl.Name)
		if e(err) {
			return errors.Wrapf(err, "Couldn't find collection '%s' to delete", cl.Name)
		}
		options := driver.SetCollectionPropertiesOptions{}
		if cl.JournalSize != nil {
			options.JournalSize = int64(*cl.JournalSize)
		}

		if cl.WaitForSync != nil {
			options.WaitForSync = cl.WaitForSync
		}
		err = col.SetProperties(ctx, options)
		return errors.Wrapf(err, "Couldn't update collection '%s'", col.Name())
	}

	return nil
}

func (g Graph) Migrate(ctx context.Context, db driver.Database, _ map[string]interface{}) error {
	switch g.Action {
	case CREATE:
		options := driver.CreateGraphOptions{}

		// Only set smart if we know the user set something.
		if g.Smart != nil {
			options.IsSmart = *g.Smart
		}
		options.SmartGraphAttribute = g.SmartGraphAttribute

		// Set the number of shards.
		numShards := 1
		if g.Shards > 0 {
			numShards = g.Shards
		}
		options.NumberOfShards = numShards

		// Map the edge collections.
		for _, ed := range g.EdgeDefinitions {
			options.EdgeDefinitions = append(
				options.EdgeDefinitions,
				driver.EdgeDefinition{
					Collection: ed.Collection,
					To:         ed.To,
					From:       ed.From,
				})
		}

		// Map the Orphan Vertices
		options.OrphanVertexCollections = g.OrphanVertices

		_, err := db.CreateGraphV2(ctx, g.Name, &options)
		return err
	case DELETE:
		aG, err := db.Graph(ctx, g.Name)
		if e(err) {
			return errors.Wrapf(err, "Couldn't find graph with name %s. Can't delete.", g.Name)
		}
		err = aG.Remove(ctx)
		if !e(err) {
			log.Printf("Deleted graph '%s'\n", g.Name)
		}
		return errors.Wrapf(err, "Couldn't remove graph %s", g.Name)
	case MODIFY:
		aG, err := db.Graph(ctx, g.Name)
		if e(err) {
			return errors.Wrapf(err, "Couldn't find graph with name %s. Can't modify.", g.Name)
		}

		// Order matters. If an edge and a vertex are removed, the edge has to
		// go first, otherwise Arango will throw an error.
		if len(g.RemoveEdges) > 0 {
			for _, re := range g.RemoveEdges {
				ec, _, err := aG.EdgeCollection(ctx, re)
				if driver.IsNotFoundGeneral(err) {
					log.Printf("Couldn't find edge collection '%s' to remove.\n", re)
					continue
				}

				if err = ec.Remove(ctx); e(err) {
					return errors.Wrapf(err, "Couldn't remove edge collection '%s'", re)
				}
			}
		}

		if len(g.RemoveVertices) > 0 {
			for _, v := range g.RemoveVertices {
				vc, err := aG.VertexCollection(ctx, v)
				if driver.IsNotFoundGeneral(err) {
					log.Printf("Couldn't find vertex '%s' to remove.", v)
				}
				if err = vc.Remove(ctx); e(err) {
					return errors.Wrapf(err, "Couldn't remove vertex collection '%s'", v)
				}

			}
		}

		if len(g.OrphanVertices) > 0 {
			for _, o := range g.OrphanVertices {
				_, err := aG.CreateVertexCollection(ctx, o)
				if e(err) {
					return errors.Wrapf(err, "Couldn't add orphan vertex '%s'", o)
				}
			}
		}

		if len(g.EdgeDefinitions) > 0 {
			for i, ed := range g.EdgeDefinitions {
				if exists, err := aG.EdgeCollectionExists(ctx, ed.Collection); exists && !e(err) {
					// Assume an update
					constraints := driver.VertexConstraints{
						From: ed.From,
						To:   ed.To,
					}
					return errors.Wrapf(
						aG.SetVertexConstraints(ctx, ed.Collection, constraints),
						"Couldn't update edge constrain #%d",
						i,
					)
				} else if !exists && !e(err) {
					vc := driver.VertexConstraints{}
					vc.From = ed.From
					vc.To = ed.To
					_, err = aG.CreateEdgeCollection(ctx, ed.Collection, vc)
					if e(err) {
						return errors.Wrapf(err, "Couldn't create edge collection '%s'", ed.Collection)
					}
				} else {
					return errors.WithStack(err)
				}
			}
		}
		return nil
	default:
		return errors.Errorf("Unknown action %s", g.Action)
	}
}

func (i FullTextIndex) Migrate(ctx context.Context, db driver.Database, _ map[string]interface{}) error {
	cl, err := db.Collection(ctx, i.Collection)
	if e(err) {
		return errors.Wrapf(
			err,
			"Couldn't create full text index on collection '%s'. Collection not found",
			i.Collection,
		)
	}
	switch i.Action {
	case DELETE:
		err = dropIndex(ctx, cl, i.Name)
		return errors.Wrapf(
			err,
			"Could not drop full text index with name '%s' in collection %s",
			i.Name, i.Collection,
		)
	case CREATE:
		options := driver.EnsureFullTextIndexOptions{}
		options.MinLength = i.MinLength
		options.Name = i.Name
		options.InBackground = i.InBackground
		_, _, err = cl.EnsureFullTextIndex(ctx, i.Fields, &options)

		return errors.Wrapf(
			err,
			"Could not create full text index with fields '%s' in collection %s",
			i.Fields, i.Collection,
		)
	default:
		return errors.Errorf("Unknown action %s", i.Action)
	}
}

func (i GeoIndex) Migrate(ctx context.Context, db driver.Database, _ map[string]interface{}) error {
	cl, err := db.Collection(ctx, i.Collection)
	if e(err) {
		return errors.Wrapf(
			err,
			"Couldn't create geo index on collection '%s'. Collection not found",
			i.Collection,
		)
	}
	switch i.Action {
	case DELETE:
		err = dropIndex(ctx, cl, i.Name)
		return errors.Wrapf(
			err,
			"Could not drop geo index with name '%s' in collection %s",
			i.Name, i.Collection,
		)
	case CREATE:
		options := driver.EnsureGeoIndexOptions{}
		options.GeoJSON = i.GeoJSON
		options.Name = i.Name
		options.InBackground = i.InBackground
		_, _, err = cl.EnsureGeoIndex(ctx, i.Fields, &options)

		return errors.Wrapf(
			err,
			"Could not create geo index with fields '%s' in collection %s",
			i.Fields, i.Collection,
		)

	default:
		return errors.Errorf("Unknown action %s", i.Action)
	}
}

func (i HashIndex) Migrate(ctx context.Context, db driver.Database, _ map[string]interface{}) error {
	cl, err := db.Collection(ctx, i.Collection)

	if e(err) {
		return errors.Wrapf(
			err,
			"Couldn't create hash index on collection '%s'. Collection not found",
			i.Collection,
		)
	}
	switch i.Action {
	case DELETE:
		err = dropIndex(ctx, cl, i.Name)
		return errors.Wrapf(
			err,
			"Could not drop hash index with name '%s' in collection %s",
			i.Name, i.Collection,
		)
	case CREATE:
		options := driver.EnsureHashIndexOptions{}
		options.NoDeduplicate = i.NoDeduplicate
		options.Sparse = i.Sparse
		options.Unique = i.Unique
		options.Name = i.Name
		options.InBackground = i.InBackground
		_, _, err = cl.EnsureHashIndex(ctx, i.Fields, &options)

		return errors.Wrapf(
			err,
			"Could not create hash index with fields '%s' in collection %s",
			i.Fields, i.Collection,
		)
	default:
		return errors.Errorf("Unknown action %s", i.Action)
	}
}

func (i PersistentIndex) Migrate(ctx context.Context, db driver.Database, _ map[string]interface{}) error {
	cl, err := db.Collection(ctx, i.Collection)
	if e(err) {
		return errors.Wrapf(
			err,
			"Couldn't create persistent index on collection '%s'. Collection not found",
			i.Collection,
		)
	}
	switch i.Action {
	case DELETE:
		err = dropIndex(ctx, cl, i.Name)
		return errors.Wrapf(
			err,
			"Could not drop persistent index with name '%s' in collection %s",
			i.Name, i.Collection,
		)
	case CREATE:
		options := driver.EnsurePersistentIndexOptions{}
		options.Sparse = i.Sparse
		options.Unique = i.Unique
		options.Name = i.Name
		options.InBackground = i.InBackground
		_, _, err = cl.EnsurePersistentIndex(ctx, i.Fields, &options)

		return errors.Wrapf(
			err,
			"Could not create persistent index with fields '%s' in collection %s",
			i.Fields, i.Collection,
		)
	default:
		return errors.Errorf("Unknown action %s", i.Action)
	}
}

func (i TTLIndex) Migrate(ctx context.Context, db driver.Database, _ map[string]interface{}) error {
	cl, err := db.Collection(ctx, i.Collection)
	if e(err) {
		return errors.Wrapf(
			err,
			"Couldn't create ttl index on collection '%s'. Collection not found",
			i.Collection,
		)
	}
	switch i.Action {
	case DELETE:
		err = dropIndex(ctx, cl, i.Name)
		return errors.Wrapf(
			err,
			"Could not drop ttl index with name '%s' in collection %s",
			i.Name, i.Collection,
		)
	case CREATE:
		options := driver.EnsureTTLIndexOptions{}
		options.Name = i.Name
		options.InBackground = i.InBackground
		_, _, err = cl.EnsureTTLIndex(ctx, i.Field, i.ExpireAfter, &options)

		return errors.Wrapf(
			err,
			"Could not create ttl index with field '%s' in collection %s",
			i.Field, i.Collection,
		)
	default:
		return errors.Errorf("Unknown action %s", i.Action)
	}
}

func (i SkiplistIndex) Migrate(ctx context.Context, db driver.Database, _ map[string]interface{}) error {
	cl, err := db.Collection(ctx, i.Collection)
	if e(err) {
		return errors.Wrapf(
			err,
			"Couldn't create skiplist index on collection '%s'. Collection not found",
			i.Collection,
		)
	}
	switch i.Action {
	case DELETE:
		err = dropIndex(ctx, cl, i.Name)
		return errors.Wrapf(
			err,
			"Could not drop skiplist index with name '%s' in collection %s",
			i.Name, i.Collection,
		)
	case CREATE:
		options := driver.EnsureSkipListIndexOptions{}
		options.Sparse = i.Sparse
		options.Unique = i.Unique
		options.NoDeduplicate = i.NoDeduplicate
		options.Name = i.Name
		options.InBackground = i.InBackground
		_, _, err = cl.EnsureSkipListIndex(ctx, i.Fields, &options)

		return errors.Wrapf(
			err,
			"Could not create skiplist index with fields '%s' in collection %s",
			i.Fields, i.Collection,
		)

	default:
		return errors.Errorf("Unknown action %s", i.Action)
	}
}

func (a AQL) Migrate(ctx context.Context, db driver.Database, extras map[string]interface{}) error {
	escaped := make(map[string]interface{})
	for k, v := range a.BindVars {
		if vstr, ok := v.(string); ok {
			escaped[k] = directReplace(vstr, extras)
		} else {
			escaped[k] = v
		}

	}
	cur, err := db.Query(ctx, a.Query, escaped)
	if e(err) {
		return errors.Wrapf(err, "Couldn't execute query '%s'", a.Query)
	}
	defer func(cur driver.Cursor) {
		err := cur.Close()
		if err != nil {
			log.Printf("could not close cursor")
		}
	}(cur)
	return nil
}

func dropIndex(ctx context.Context, cl driver.Collection, name string) error {
	var exists bool
	var idx driver.Index
	var err error

	exists, err = cl.IndexExists(ctx, name)
	if e(err) {
		return errors.Wrapf(err, "Error finding index '%s'", name)
	}

	if exists {
		// get index
		idx, err = cl.Index(ctx, name)
		if e(err) {
			return errors.Wrapf(err, "Error retrieving index '%s'", name)
		}

		// drop index
		err = idx.Remove(ctx)
		if e(err) {
			return errors.Wrapf(err, "Error dropping index '%s'", name)
		}
	}

	return nil
}
