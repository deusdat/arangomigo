/*
Package main allows the tool to execute from the command line.
*/
package main

import (
	"flag"
	"fmt"
)

func main() {
	config := load()
	fmt.Printf("%+v\n", config)
}

func load() *Config {
	host := flag.String("host", "localhost:8529", "The url for the host with protocol, host and port")
	user := flag.String("user", "", "the username for root like arangodb powers")
	password := flag.String("password", "", "The password for the account with root like powers")
	useSSL := flag.Bool("useSSL", false, "Existance indicates that the connection should use TLS")
	migrationPath := flag.String("migrationsPath", "", "Location of the directory holding the migrations")
	action := flag.String("action", "up", "Indicates which migration action to use: up or down")
	
	flag.Parse()
	
	return &Config{
		host:           *host,
		user:           *user,
		password:       *password,
		ssl:            *useSSL,
		migrationsPath: *migrationPath,
		action:         *action,
	}
}

type Config struct {
	host           string
	user           string
	password       string
	ssl            bool
	migrationsPath string
	action         string
}
