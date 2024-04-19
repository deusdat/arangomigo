package arangomigo

import (
	"context"
	"log"
	"sort"
	"testing"

	driver "github.com/arangodb/go-driver"
	"github.com/stretchr/testify/assert"
)

func TestConfigLoad(t *testing.T) {
	configFile := "testdata/complete/config.yaml"
	conf, err := loadConf(configFile)
	if e(err) {
		log.Fatal(err)
	}
	if len(conf.MigrationsPath) != 1 {
		log.Fatal("Invalid migration path")
	}
}

func TestFullMigration(t *testing.T) {
	configFile := "testdata/complete/config.yaml"

	conf, err := loadConf(configFile)
	if e(err) {
		log.Fatal(err)
	}

	ctx := context.Background()

	cl, err := client(*conf)
	if e(err) {
		log.Fatal(err)
	}

	db, err := cl.Database(ctx, conf.Db)
	if err == nil {
		err := db.Remove(ctx)
		if e(err) {
			t.Fatal("Couldn't prepare for test")
		}
	}

	_, err = cl.Database(ctx, conf.Db)
	if !driver.IsNotFoundGeneral(err) {
		t.Fatal("Could not connect to the Database", err)
	}

	TriggerMigration(configFile)

	// Look to see if everything was made properly.
	db, err = cl.Database(ctx, conf.Db)
	assert.NoError(t, err, "Unable to find the database")

	recipes, err := db.Collection(ctx, "recipes")
	assert.NoError(t, err, "Could not find recipes collection")

	// Should find the custom recipe inserted by AQL.
	desiredKey := "hello"
	r := recipe{}
	md, err := recipes.ReadDocument(ctx, desiredKey, &r)
	assert.Equal(t, md.Key, desiredKey, "Meta data should match desired key.")
	assert.Equal(t, r.Key, desiredKey, "Document key should match desired key.")
	assert.Equal(t, "Lots of mayo", r.WithEscaped, "Should have updated the escaped var.")
	assert.Equal(t, "Fish", r.MeatType, "Should not have changed.")
	assert.Equal(t, "Taco Fishy", r.Name)

	// Can't really tell which indexes are available, just that recipes should have
	// 7: 1 for the PK and 6 others.
	idxs, err := recipes.Indexes(ctx)
	assert.Equal(t, 9, len(idxs), "Recipes should have 9 indexes")

	// Make sure wait for sync sticks.
	colprop, err := recipes.Properties(ctx)
	assert.True(t, colprop.WaitForSync, "Should wait for sync.")

	g, err := db.Graph(ctx, "testing_graph")
	assert.NoError(t, err, "Should have gotten a graph")
	vcs, err := g.VertexCollections(ctx)
	assert.NoError(t, err, "Should have gotten vertices")

	viewExists, viewExistsErr := db.ViewExists(ctx, "testing_searchalias_view")
	assert.NoError(t, viewExistsErr, "Should have got a search-alias view")
	assert.True(t, viewExists, "Should have got testing_searchalias_view")

	view, err := db.View(ctx, "testing_searchalias_view")
	assert.NoError(t, err, "Should have got view")
	v, err := view.ArangoSearchViewAlias()
	assert.NoError(t, err, "Should have got view as search-alias view")
	viewProperties, err := v.Properties(ctx)
	assert.NoError(t, err, "Should have got search-alias view properties")
	assert.Len(t, viewProperties.Indexes, 1, "Should have 1 index")

	// Vertices include those in edges and oraphans.
	justNames := []string{}
	for _, k := range vcs {
		justNames = append(justNames, k.Name())
	}
	sort.Strings(justNames)

	assert.Equal(t, []string{"another", "recipes", "users"}, justNames)
}

func TestMultiPathMigration(t *testing.T) {
	configFile := "testdata/complete2/config.yaml"

	conf, err := loadConf(configFile)
	if e(err) {
		log.Fatal(err)
	}

	ctx := context.Background()

	cl, err := client(*conf)
	if e(err) {
		log.Fatal(err)
	}

	db, err := cl.Database(ctx, conf.Db)
	if err == nil {
		err := db.Remove(ctx)
		if e(err) {
			t.Fatal("Couldn't prepare for test")
		}
	}

	_, err = cl.Database(ctx, conf.Db)
	if !driver.IsNotFound(err) {
		t.Fatal("Could not connect to the Database", err)
	}

	TriggerMigration(configFile)

	// Look to see if everything was made properly.
	db, err = cl.Database(ctx, conf.Db)
	assert.NoError(t, err, "Unable to find the database")

	recipes, err := db.Collection(ctx, "Recipes")
	assert.NoError(t, err, "Could not find recipes collection")

	// Should find the custom recipe inserted by AQL.

	// Can't really tell which indexes are available, just that recipes should
	// have just 1 for the PK.
	idxs, err := recipes.Indexes(ctx)
	assert.Equal(t, 1, len(idxs), "Recipes should have 1 index")

	// Make sure wait for sync sticks.
	colprop, err := recipes.Properties(ctx)
	assert.NotNil(t, colprop, "Should have the collection properties")
	assert.False(t, colprop.WaitForSync, "Should wait for sync.")
}

func TestSkipSSLVerify(t *testing.T) {
	configFile := "testdata/skip_ssl_verify/config.yaml"

	conf, err := loadConf(configFile)
	if e(err) {
		log.Fatal(err)
	}

	ctx := context.Background()

	cl, err := client(*conf)
	if e(err) {
		log.Fatal(err)
	}

	db, err := cl.Database(ctx, conf.Db)
	if err == nil {
		err := db.Remove(ctx)
		if e(err) {
			t.Fatal("Couldn't prepare for test")
		}
	}

	_, err = cl.Database(ctx, conf.Db)
	if !driver.IsNotFound(err) {
		t.Fatal("Could not connect to the Database", err)
	}

	TriggerMigration(configFile)

	// Look to see if everything was made properly.
	db, err = cl.Database(ctx, conf.Db)
	assert.NoError(t, err, "Unable to find the database")
}

type recipe struct {
	Name        string
	WithEscaped string
	MeatType    string
	Key         string `json:"_key"`
}
