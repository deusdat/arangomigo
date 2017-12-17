package main

import (
	"github.com/solher/arangolite"
	"regexp"
)

type Migration interface {
	migrate(db *arangolite.Database) error
	FileName() string
	SetFileName(name string)
}

type Operation struct {
	fileName string
	Type     string
	Name     string
	Action   Action
}

type Action string

const (
	CREATE Action = "create"
	DELETE Action = "delete"
	MODIFY Action = "modify"
	RUN    Action = "run"
)

// Declares the various patterns for mapping the types.
var collection = regexp.MustCompile(`^type:\scollection\n`)

type User struct {
	username string
	password string
}

type Database struct {
	Operation `yaml:",inline"`

	Allowed    []User
	Disallowed []string
}

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

func (cl Collection) migrate(db *arangolite.Database) error {
	switch cl.Action {
	case DELETE:
	}
	return nil
}

func (cl Collection) FileName() string {
	return cl.fileName
}

func (cl *Collection) SetFileName(fileName string) {
	cl.fileName = fileName
}
