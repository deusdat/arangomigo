package main

import (
	"fmt"
	//"gopkg.in/yaml.v2"
	//"log"
	"testing"
)

func TestLoadFromPath(t *testing.T) {
	ops := loadFrom("testdata/simple_migrations")
	if len(ops) != 2 {
		t.Error("Should have loaded only two files")
	}

	assertEqual(t, ops[0].FileName(), "1.up.migration", "")
	fmt.Printf("%v", ops)
}

func assertEqual(t *testing.T, a interface{}, b interface{}, message string) {
	if a == b {
		return
	}
	if len(message) == 0 {
		message = fmt.Sprintf("%v != %v", a, b)
	}
	t.Fatal(message)
}
