package main

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadFromPath(t *testing.T) {
	assert.Panics(
		t,
		func() { loadFrom("testdata/simple_migrations") },
		"Should have paniced")

}

func TestSimpleDataVersion(t *testing.T) {
	var datebased = []string{
		"2017.02.0.1.migration",
		"2017.01.02.migration",
		"2017.05.09.migration",
	}

	sort.Slice(datebased, nearlyLexical(datebased))
	assert.Equal(t, []string{
		"2017.01.02.migration",
		"2017.02.0.1.migration",
		"2017.05.09.migration",
	}, datebased)
}

func TestDateAndDescription(t *testing.T) {
	dateAndComments := []string{
		"2.01_Add_a_new_collection.migration",
		"1.02_Description_Here.migration",
		"5.09_They_keep_pulling_me_in.migration",
	}
	sort.Slice(dateAndComments, nearlyLexical(dateAndComments))
	assert.Equal(
		t,
		[]string{
			"1.02_Description_Here.migration",
			"2.01_Add_a_new_collection.migration",
			"5.09_They_keep_pulling_me_in.migration"},
		dateAndComments,
	)
}

func TestNumericVersions(t *testing.T) {
	v := []string{"1.migration", "12.migration", "2.migration"}
	sort.Slice(v, nearlyLexical(v))
	assert.Equal(
		t,
		[]string{"1.migration", "2.migration", "12.migration"},
		v,
	)
}

func TestDecNumericVersions(t *testing.T) {
	v := []string{"1.0.migration", "12.migration", "1.2.migration"}
	sort.Slice(v, nearlyLexical(v))
	assert.Equal(
		t,
		[]string{"1.0.migration", "1.2.migration", "12.migration"},
		v,
	)
}
