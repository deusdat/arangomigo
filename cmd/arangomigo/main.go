/*
Package main allows the tool to execute from the command line.
*/
package main

import (
	"log"
	"os"

	"github.com/deusdat/arangomigo"
)

func main() {
	configAt := ""
	if len(os.Args) > 1 {
		configAt = os.Args[1]
	} else {
		log.Fatal("Please specify the path for the configuration")
	}

	arangomigo.TriggerMigration(configAt)
}
