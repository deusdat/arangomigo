package main

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"testing"
)

func TestLoadFromPath(t *testing.T) {
	ops, err := loadFrom("testdata/simple_migrations")
	if err != nil {
		t.Error("Couldn't start test %s", err)
	}
	if len(ops) != 2 {
		t.Error("Should have loaded only two files")
	}

	assertEqual(t, ops[0].FileName(), "1.up.migration", "")
	fmt.Printf("%v\n", ops)
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

func TestCreateYaml(t *testing.T) {
	type A struct {
		B string
		C string
	}

	type D struct {
		A
		E string
	}

	in := D{A: A{B: "Hello", C: "World"}, E: "Goodbye"}
	out, _ := yaml.Marshal(&in)

	fmt.Printf("---- out dump:\n%s\n", string(out))
}
