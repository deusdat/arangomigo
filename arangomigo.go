/*
Package main allows the tool to execute from the command line.
*/
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

const (
	waitOnDb  = 0
	defaultDb = "_system"
)

func main() {
	configAt := ""
	if len(os.Args) > 1 {
		configAt = os.Args[1]
	} else {
		log.Fatal("Please specify the path for the configuration")
	}

	triggerMigration(configAt)
}

func triggerMigration(configAt string) {
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
	return perform(ctx, c)
}

// Reads in a yaml file at the confLoc and returns the Config instance.
func loadConf(confLoc string) (*Config, error) {
	bytes, _, err := open(confLoc)
	if e(err) {
		return nil, fmt.Errorf("Couldn't locate configation at path '%s'", confLoc)
	}

	conf := Config{}
	err = yaml.UnmarshalStrict(bytes, &conf)
	if e(err) {
		return nil, errors.Wrapf(err, "Couldn't parse configation at path '%s'", confLoc)
	}

	if conf.Db == "" {
		return nil, errors.New("Please specifiy the database name in the config")
	}
	encased := make(map[string]string)
	for k, v := range conf.Extras {
		encased[fmt.Sprintf("${%s}", k)] = v
	}
	conf.Extras = encased
	return &conf, nil
}

// Config The content of a migration configuration.
type Config struct {
	Endpoints      []string
	Username       string
	Password       string
	MigrationsPath string
	Db             string
	// Extras allows the user to pass in replaced variables
	Extras map[string]string
}
