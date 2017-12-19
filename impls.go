package main

import (
	"context"
	"fmt"
	"github.com/solher/arangolite"
	"github.com/solher/arangolite/requests"
	"regexp"
)

type Migration interface {
	migrate(ctx context.Context, db *arangolite.Database) error
	FileName() string
	SetFileName(name string)
	CheckSum() string
	SetCheckSum(sum string)
}

type Operation struct {
	checksum string
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
var database = regexp.MustCompile(`^type:\sdatabase\n`)

type User struct {
	Username string
	Password string
}

type Database struct {
	Operation `yaml:",inline"`

	Allowed    []User
	Disallowed []string
}

func (d Database) migrate(ctx context.Context, db *arangolite.Database) error {
	fmt.Println("Should have done a migration like thing")
	var resultErr error = nil
	switch d.Action {
	case CREATE:

		rdb := requests.CreateDatabase{}

		rdb.Name = d.Name

		if len(d.Allowed) > 0 {
			um := make([]map[string]interface{}, len(d.Allowed))
			rdb.Users = um
			for i, u := range d.Allowed {
				jsonUser := map[string]interface{}{
					"username": u.Username,
					"passwd":   u.Password,
				}

				um[i] = jsonUser
			}
		}

		resultErr = db.Run(ctx, nil, &rdb)

	}
	return resultErr
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

func (cl Collection) migrate(ctx context.Context, db *arangolite.Database) error {
	switch cl.Action {
	case DELETE:
	}
	return nil
}

func (cl *Operation) FileName() string {
	return cl.fileName
}

func (cl *Operation) SetFileName(fileName string) {
	cl.fileName = fileName
}

func (cl *Operation) CheckSum() string {
	return cl.checksum
}

func (cl *Operation) SetCheckSum(sum string) {
	cl.checksum = sum
}
