package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	driver "github.com/arangodb/go-driver"
)

type MockDB struct {
	driver.Database
}

func (mdb MockDB) Collection(ctx context.Context, name string) (driver.Collection, error) {
	col := MockCol{name: "FakeIndex"}

	return col, nil
}

type MockCol struct {
	driver.Collection
	name string
}

func (mc MockCol) Name() string {
	return mc.name
}

func TestHashRejectDelete(t *testing.T) {
	hi := HashIndex{}
	hi.Action = DELETE

	//assert.Panics(t, func() {
	err := hi.migrate(
		context.Background(),
		MockDB{},
		nil,
	)
	//})
	assert.EqualError(t, err, "Due to Arango API limitations, you cannot delete an index")
}

func TestFullTextRejectDelete(t *testing.T) {
	hi := FullTextIndex{}
	hi.Action = DELETE

	//assert.Panics(t, func() {
	err := hi.migrate(
		context.Background(),
		MockDB{},
		nil,
	)
	//})
	assert.EqualError(t, err, "Due to Arango API limitations, you cannot delete an index")
}

func TestGeoRejectDelete(t *testing.T) {
	hi := GeoIndex{}
	hi.Action = DELETE

	//assert.Panics(t, func() {
	err := hi.migrate(
		context.Background(),
		MockDB{},
		nil,
	)
	//})
	assert.EqualError(t, err, "Due to Arango API limitations, you cannot delete an index")
}

func TestPersistentRejectDelete(t *testing.T) {
	hi := PersistentIndex{}
	hi.Action = DELETE

	//assert.Panics(t, func() {
	err := hi.migrate(
		context.Background(),
		MockDB{},
		nil,
	)
	//})
	assert.EqualError(t, err, "Due to Arango API limitations, you cannot delete an index")
}
func TestSkipListRejectDelete(t *testing.T) {
	hi := SkiplistIndex{}
	hi.Action = DELETE

	//assert.Panics(t, func() {
	err := hi.migrate(
		context.Background(),
		MockDB{},
		nil,
	)
	//})
	assert.EqualError(t, err, "Due to Arango API limitations, you cannot delete an index")
}
