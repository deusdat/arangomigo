/*
Package main allows the tool to execute from the command line.
*/
package main

import (
	"context"
	"errors"
	"fmt"
	"gopkg.in/yaml.v2"
	"log"
	"os"
)

const (
	WAIT_ON_NO_DB = 0
	DEFAULT_DB    = "_system"
)

func main() {
	configAt := ""
	if len(os.Args) > 1 {
		configAt = os.Args[1]
	} else {
		log.Fatal("Please specify the path for the configuration")
	}

	config, err := loadConf(configAt)
	if e(err) {
		log.Fatal(err)
	}

	if err := migrate(*config); err != nil {
		log.Fatal("Could not perform migration\n", err)
	}
	fmt.Println("Successfully completed migration")
}

// TODO remember that having replayable migrations need to be possible too.
// Have branch those into running at the end.

func migrate(c Config) error {
	ctx := context.Background()

	if err := perform(ctx, c); err != nil {
		return err
	}
	return nil
}

// Reads in a yaml file at the confLoc and returns the Config instance.
func loadConf(confLoc string) (*Config, error) {
	bytes, _, err := open(confLoc)
	if e(err) {
		return nil, errors.New(fmt.Sprintf("Couldn't locate configation at path '%s'", confLoc))
	}

	conf := Config{}
	err = yaml.UnmarshalStrict(bytes, &conf)
	if e(err) {
		return nil, errors.New(fmt.Sprintf("Couldn't parse configation at path '%s'", confLoc, err))
	}

	if conf.Db == "" {
		return nil, errors.New("Please specifiy the database name in the config")
	}
	return &conf, nil
}

type Config struct {
	Endpoints      []string
	Username       string
	Password       string
	MigrationsPath string
	Db             string
}
