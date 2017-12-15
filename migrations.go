package main

import (
	"errors"
	"fmt"
	"github.com/solher/arangolite"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"path/filepath"
	//"reflect"
	"regexp"
	"sort"
)

/*
What does this module need to do?
 - Need a way to find all of the files with a certain file name pattern: migration*.yaml
 - Need to load the files into a structure that matches the yaml format.
 - Needs to return the whole list/array of structs to the caller.
*/

type Migration interface {
	migrate(action Action, db *arangolite.Database) error
	FileName() string
	SetFileName(name string)
}

type Action string

const (
	CREATE Action = "create"
	DELETE Action = "delete"
	MODIFY Action = "modify"
)

// Declares the various patterns for mapping the types.
var collection = regexp.MustCompile(`^type:\scollection\n`)

type Collection struct {
	fileName       string
	Type           string
	Name           string
	Action         Action
	ShardKeys      []string
	JournalSize    int
	NumberOfShards int
	WaitForSync    bool
	AllowUserKeys  bool
	Volatile       bool
	Compactable    bool
}

func (this *Collection) migrate(action Action, db *arangolite.Database) error {
	return nil
}

func (this *Collection) FileName() string {
	return this.fileName
}

func (this *Collection) SetFileName(fileName string) {
	this.fileName = fileName
}

func loadFrom(path string) []Migration {
	parentDir := filepath.Join(path, "*.migration")
	migrations, err := filepath.Glob(parentDir)

	// This will destroy the whole process.
	if err != nil {
		log.Fatal(err)
	}
	sort.Strings(migrations)

	var answer []Migration
	for _, migration := range migrations {
		fmt.Printf("file name: %s\n", migration)
		as := toStruct(migration)
		fmt.Printf("The migration is %+v\n", as)
		answer = append(answer, as)
	}

	return answer
}

// Opens the path into a byte slice.
func open(childPath string) []byte {
	bytes, err := ioutil.ReadFile(childPath)
	if err != nil {
		log.Fatal(err)
	}
	return bytes
}

func pickT(contents []byte) (Migration, error) {
	s := string(contents)
	switch {
	case collection.MatchString(s):
		return new(Collection), nil
	default:
		return nil, errors.New("Can't determine YAML type")
	}
}

func toStruct(childPath string) Migration {
	contents := open(childPath)

	t, err := pickT(contents)
	if err != nil {
		log.Fatal(err)
	}

	//c := Collection{}
	err = yaml.UnmarshalStrict(contents, t)
	if err != nil {
		log.Fatal(err)
	}

	t.SetFileName(filepath.Base(childPath))
	return t
}
