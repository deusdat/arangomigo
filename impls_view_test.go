package main

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"testing"

	driver "github.com/arangodb/go-driver"
)

// Simple struct that mocks Arango's View interface.
type MockView struct {
	driver.View
	name string
}

// Implements View.Name() and used in testing.
func (mv MockView) Name() string {
	return mv.name
}

// Implements View.ArangoSearchView() to get an ArangoSearchView or return an error as the test expects.
func (mv MockView) ArangoSearchView() (driver.ArangoSearchView, error) {
	if "GetSearchViewError" == mv.Name() {
		return nil, errors.New("this is just a test")

	}
	return MockArangoSearchView{
		name:mv.Name(),
	}, nil
}

// Implements View.Remove.
func (mv MockView) Remove(ctx context.Context) error {
	return errors.New("this is just a test")
}

// Simple struct that implements Arango's ArangoSearchView interface.
type MockArangoSearchView struct {
	name string
}

// Implements ArangoSearchView.Name()
func (mv MockArangoSearchView) Name() string {
	return mv.name
}

// Implements ArangoSearchView.Type()
func (mv MockArangoSearchView) Type() driver.ViewType {
	return driver.ViewTypeArangoSearch
}

// Implements ArangoSearchView.Properties()
func (mv MockArangoSearchView) Properties(ctx context.Context) (driver.ArangoSearchViewProperties, error) {
	return driver.ArangoSearchViewProperties{}, nil
}

// An empty implementation of ArangoSearchView.View.ArangoSearchView().
func (mv MockArangoSearchView) ArangoSearchView() (driver.ArangoSearchView, error) {
	return nil, nil
}

// An empty implementation of ArangoSearchView.View.Database().
func (mv MockArangoSearchView) Database() driver.Database {
	return nil
}

// An empty implementation of ArangoSearchView.View.Remove().
func (mv MockArangoSearchView) Remove(ctx context.Context) error {
	return nil
}

// Implements ArangoSearchView.SetProperties()
func (mv MockArangoSearchView) SetProperties(ctx context.Context, options driver.ArangoSearchViewProperties) error {
	return errors.New("this is just a test")
}

// Implements Database.CreateArangoSearchView()
func (mdb MockDB) CreateArangoSearchView(ctx context.Context, name string, options *driver.ArangoSearchViewProperties) (driver.ArangoSearchView, error) {
	return nil, errors.New("this is just a test")
}

// Implements Database.View() to get a View or return an error as the test expects.
func (mdb MockDB) View(ctx context.Context, name string) (driver.View, error) {
	if name == "ViewError" {
		return nil, errors.New("this is just a test")

	}
	return MockView{
		name: name,
	}, nil
}

// Test the error is handled during creating a view.
func TestViewCreateError(t *testing.T) {

	view := SearchView{}
	view.Action = CREATE
	view.Name = "TestView"

	err := view.migrate(
		context.Background(),
		MockDB{},
		nil)

	assert.EqualError(t, err, "Couldn't create view 'TestView': this is just a test")
}

// Test the error is handled when getting a view for delete.
func TestViewDeleteErrorGetting(t *testing.T) {
	view := SearchView{}
	view.Action = DELETE
	view.Name = "ViewError"

	err := view.migrate(
		context.Background(),
		MockDB{},
		nil)

	assert.EqualError(t, err, "Couldn't find view 'ViewError' to delete: this is just a test")
}

// Test the error is handled during deleting a view.
func TestViewDeleteErrorRemove(t *testing.T) {
	view := SearchView{}
	view.Action = DELETE
	view.Name = "TestView"

	err := view.migrate(
		context.Background(),
		MockDB{},
		nil)

	assert.EqualError(t, err, "Couldn't delete view 'TestView': this is just a test")
}

// Test the error is handled when getting a view to update.
func TestViewModifyErrorGetting(t *testing.T) {
	view := SearchView{}
	view.Action = MODIFY
	view.Name = "ViewError"

	err := view.migrate(
		context.Background(),
		MockDB{},
		nil)

	assert.EqualError(t, err, "Couldn't find view 'ViewError' to update: this is just a test")
}

// Test the error is handled when getting the ArangoSearchView from a view.
func TestViewModifyErrorGetSearchView(t *testing.T) {
	view := SearchView{}
	view.Action = MODIFY
	view.Name = "GetSearchViewError"

	err := view.migrate(
		context.Background(),
		MockDB{},
		nil)

	assert.EqualError(t, err, "Couldn't get ArangoSearchView 'GetSearchViewError' to update: this is just a test")
}

// Test the error is handled when updating the ArangoSearchView's properties.
func TestViewModifyErrorSetProperties(t *testing.T) {
	view := SearchView{}
	view.Action = MODIFY
	view.Name = "TestView"

	err := view.migrate(
		context.Background(),
		MockDB{},
		nil)

	assert.EqualError(t, err, "Couldn't update SearchView 'TestView': this is just a test")
}