package arangomigo

import (
	"context"
	"os"
	"testing"
)

func GetEnv(key string, fallback string) string {
	value, exists := os.LookupEnv(key)
	if exists {
		return value
	}
	return fallback
}

func TestPerform(t *testing.T) {
	arangoUrl := GetEnv("ARANGO_URL", "http://0.0.0.0:8529")
	config := Config{
		Endpoints: []string{arangoUrl},
		Username:  "root",
		Password:  "simple",
		Db:        "MigoFullPerform",
		Extras: map[string]interface{}{
			"patricksUser":     "jdavenpo",
			"patricksPassword": "Extrem!Password&^%$#",
			"shouldBeANumber":  "10",
			"secret":           "Lots of mayo",
		},
	}

	journalSize := 10485760
	waitForSync := true

	migrations := []Migration{
		&Database{
			Operation: Operation{Type: "database", Action: CREATE, Name: "MigoFullPerform"},
			Allowed:   []User{{Username: "patrick", Password: "secret"}},
		},
		&Collection{
			Operation:   Operation{Type: "collection", Action: CREATE, Name: "recipes"},
			JournalSize: &journalSize,
			WaitForSync: &waitForSync,
		},
		&AQL{
			Operation: Operation{Type: "aql"},
			Query:     `INSERT {Name: "Taco Fishy", WithEscaped: @escaped, MeatType: @meat, _key: "hello"} IN recipes`,
			BindVars:  map[string]interface{}{"escaped": "secret", "meat": "Fish"},
		},
		&Graph{
			Operation:       Operation{Type: "graph", Action: CREATE, Name: "testing_graph"},
			EdgeDefinitions: []EdgeDefinition{{Collection: "relationships", From: []string{"recipes"}, To: []string{"recipes"}}},
		},
	}

	err := PerformMigrations(context.Background(), config, migrations)
	if e(err) {
		t.Fatal(err)
	}
}
