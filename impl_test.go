package arangomigo

import (
	"context"
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
