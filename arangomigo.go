/*
Package arangomigo allows the tool to execute from the command line.
*/
package arangomigo

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"log"
)

func TriggerMigration(configAt string) {
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

	pm, err := migrations(c.MigrationsPath)
	if e(err) {
		return err
	}

	return perform(ctx, c, pm)
}

// Reads in a yaml file at the confLoc and returns the Config instance.
func loadConf(confLoc string) (*Config, error) {
	bytes, _, err := open(confLoc)
	if e(err) {
		return nil, fmt.Errorf("Couldn't locate configation at path '%s'", confLoc)
	}

	conf := Config{}
	err = yaml.Unmarshal(bytes, &conf)
	if e(err) {
		return nil, errors.Wrapf(err, "Couldn't parse configation at path '%s'", confLoc)
	}

	if conf.Db == "" {
		return nil, errors.New("Please specifiy the database name in the config")
	}
	encased := make(map[string]interface{})
	for k, v := range conf.Extras {
		encased[fmt.Sprintf("${%s}", k)] = v
	}
	conf.Extras = encased
	return &conf, nil
}

type StringArray []string

func (a *StringArray) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var multi []string
	err := unmarshal(&multi)
	if err != nil {
		var single string
		err = unmarshal(&single)
		if err != nil {
			return err
		}
		*a = []string{single}
	} else {
		*a = multi
	}
	return nil
}

// Config The content of a migration configuration.
type Config struct {
	Endpoints      []string
	Username       string
	Password       string
	MigrationsPath StringArray
	Db             string
	// Extras allows the user to pass in replaced variables
	Extras map[string]interface{}
}
