package main

import (
	"context"
	"log"
	"testing"

	driver "github.com/arangodb/go-driver"
)

func TestFullMigration(t *testing.T) {
	configFile := "testdata/complete/config.yaml"

	conf, err := loadConf(configFile)
	if e(err) {
		log.Fatal(err)
	}

	ctx := context.Background()

	cl, err := client(ctx, *conf)
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
		t.Fatal("Database should not be there")
	}

	triggerMigration(configFile)
}
