package main

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"sort"

	//driver "github.com/arangodb/go-driver" // This pisses me off. Why expose it?
	driver "github.com/arangodb/go-driver"
	"gopkg.in/yaml.v2"
)

// Operation the common elements for all migrations.
type Operation struct {
	checksum string
	fileName string
	Type     string
	Name     string
	Action   Action
}

// Action enumerated values for valid operation actions.
type Action string

// Enumerated values for the Action
const (
	CREATE Action = "create"
	DELETE Action = "delete"
	MODIFY Action = "modify"
	RUN    Action = "run"
)

// Declares the various patterns for mapping the types.
var collection = regexp.MustCompile(`^type:\scollection`)
var database = regexp.MustCompile(`^type:\sdatabase`)
var graph = regexp.MustCompile(`^type:\sgraph`)

// User the data used to update a user account
type User struct {
	Username string
	Password string
}

// Database the YAML struct for configuring a database migration.
type Database struct {
	Operation `yaml:",inline"`

	Allowed    []User
	Disallowed []string

	cl driver.Client
	db driver.Database
}

// Collection the YAML struct for configuring a collection migration.
type Collection struct {
	Operation `yaml:",inline"`

	ShardKeys      []string
	JournalSize    int
	NumberOfShards int
	WaitForSync    bool
	AllowUserKeys  bool
	Volatile       bool
	Compactable    bool
}

// EdgeDefinition contains all information needed to define
// a single edge in a graph.
type EdgeDefinition struct {
	// The name of the edge collection to be used.
	Collection string `json:"collection"`
	// To contains the names of one or more edge collections that can contain target vertices.
	To []string `json:"to"`
	// From contains the names of one or more vertex collections that can contain source vertices.
	From []string `json:"from"`
}

// Graph allows a user to manage graphs
type Graph struct {
	Operation `yaml:",inline"`
	// Smart indicates that the graph uses the Enterprise
	// edition's graph management.
	Smart bool
	// SmartGraphAttribute is the attribute used to shuffle vertexes.
	SmartGraphAttribute string
	// Shards is the number of shards each collection has.
	Shards int
	// OrphanVertex
	OrphanVertex []string
	// EdgeDifinition creates a single edge between vertexes
	EdgeDefinitions []EdgeDefinition
}

// PairedMigrations Defines the primary change and an undo operation if provided.
// Presently undo is not a supported feature. After reading Flyway's
// history of the feature, it might  never be supported
type PairedMigrations struct {
	change Migration
	undo   Migration
}

// Pairs migrations together.
// Returns an error if unable to find migrations.
func migrations(path string) ([]PairedMigrations, error) {
	migrations, err := loadFrom(path)
	if err != nil {
		return nil, err
	}
	if len(migrations) == 0 {
		return nil, errors.New("Could not find migrations at path '" + path + "'")
	}
	var pms []PairedMigrations

	for _, m := range migrations {
		pm := PairedMigrations{change: m, undo: nil}
		pms = append(pms, pm)
	}

	return pms, nil
}

// Loads a set of migrations from a given directory.
func loadFrom(path string) ([]Migration, error) {
	parentDir := filepath.Join(path, "*.migration")
	migrations, err := filepath.Glob(parentDir)

	// This will destroy the whole process.
	if err != nil {
		return nil, err
	}

	sort.Strings(migrations)

	var answer []Migration
	for _, migration := range migrations {
		fmt.Printf("file name: %s\n", migration)
		as, err := toStruct(migration)
		if err != nil {
			return answer, err
		}
		fmt.Printf("The migration is %+v\n", as)
		answer = append(answer, as)
	}

	return answer, nil
}

// Opens the path into a byte slice.
// Returns the bytes, the file's checksum, and an error.
func open(childPath string) ([]byte, string, error) {
	bytes, err := ioutil.ReadFile(childPath)
	if err != nil {
		return nil, "", err
	}

	chk := md5.Sum(bytes)
	return bytes, hex.EncodeToString(chk[:]), nil
}

// Reads the migration contents to pick the proper type.
func pickT(contents []byte) (Migration, error) {
	s := string(contents)
	switch {
	case collection.MatchString(s):
		return new(Collection), nil
	case database.MatchString(s):
		return new(Database), nil
	case graph.MatchString(s):
		return new(Graph), nil
	default:
		return nil, errors.New("Can't determine YAML type")
	}
}

/*
	Converts a path to the proper underlying types specified in
	the childPath.
*/
func toStruct(childPath string) (Migration, error) {
	contents, checksum, err := open(childPath)

	t, err := pickT(contents)
	if err != nil {
		return nil, err
	}

	err = yaml.UnmarshalStrict(contents, t)
	if err != nil {
		return nil, err
	}

	t.SetFileName(filepath.Base(childPath))
	t.SetCheckSum(checksum)
	return t, nil
}
