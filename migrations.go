package main

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	//driver "github.com/arangodb/go-driver" // This pisses me off. Why expose it?
	driver "github.com/arangodb/go-driver"
	"gopkg.in/yaml.v2"
)

// Operation the common elements for all migrations.
type Operation struct {
	checksum string
	fileName string
	Type     string
	Name     string
	Action   Action
}

// Action enumerated values for valid operation actions.
type Action string

// Enumerated values for the Action
const (
	CREATE Action = "create"
	DELETE Action = "delete"
	MODIFY Action = "modify"
	RUN    Action = "run"
)

// Declares the various patterns for mapping the types.
var collection = regexp.MustCompile(`^type:\scollection`)
var database = regexp.MustCompile(`^type:\sdatabase`)
var graph = regexp.MustCompile(`^type:\sgraph`)
var aql = regexp.MustCompile(`^type:\saql`)
var fulltextidx = regexp.MustCompile(`^type:\sfulltextindex`)
var geoidx = regexp.MustCompile(`^type:\sgeoindex`)
var hashidx = regexp.MustCompile(`^type:\shashindex`)
var persistentidx = regexp.MustCompile(`^type:\spersistentindex`)
var skipidx = regexp.MustCompile(`^type:\sskiplistindex`)
var view = regexp.MustCompile(`^type:\sview`)

// User the data used to update a user account
type User struct {
	Username string
	Password string
}

// Database the YAML struct for configuring a database migration.
type Database struct {
	Operation `yaml:",inline"`

	Allowed    []User
	Disallowed []string

	cl driver.Client
	db driver.Database
}

// Collection the YAML struct for configuring a collection migration.
type Collection struct {
	Operation `yaml:",inline"`

	ShardKeys      *[]string
	JournalSize    *int
	NumberOfShards *int
	WaitForSync    *bool
	AllowUserKeys  *bool
	Volatile       *bool
	Compactable    *bool
}

// FullTextIndex defines how to build a full text index on a field
type FullTextIndex struct {
	Operation  `yaml:",inline"`
	Fields     []string
	Collection string
	MinLength  int
}

// GeoIndex creates a GeoIndex within the specified collection.
type GeoIndex struct {
	Operation  `yaml:",inline"`
	Fields     []string
	Collection string
	GeoJSON    bool
}

// HashIndex creates a hash index on the fields within the specified Collection.
type HashIndex struct {
	Operation     `yaml:",inline"`
	Fields        []string
	Collection    string
	Unique        bool
	Sparse        bool
	NoDeduplicate bool
}

// PersistentIndex creates a persistent index on the collections' fields.
type PersistentIndex struct {
	Operation  `yaml:",inline"`
	Fields     []string
	Collection string
	Unique     bool
	Sparse     bool
}

// SkiplistIndex creates a sliplist index on the collections' fields.
type SkiplistIndex struct {
	Operation     `yaml:",inline"`
	Fields        []string
	Collection    string
	Unique        bool
	Sparse        bool
	NoDeduplicate bool
}

// AQL allows arbitrary AQL execution as part of the migration.
type AQL struct {
	Operation `yaml:",inline"`
	Query     string
	BindVars  map[string]interface{}
}

// EdgeDefinition contains all information needed to define
// a single edge in a graph.
type EdgeDefinition struct {
	// The name of the edge collection to be used.
	Collection string `json:"collection"`
	// To contains the names of one or more edge collections that can contain target vertices.
	To []string `json:"to"`
	// From contains the names of one or more vertex collections that can contain source vertices.
	From []string `json:"from"`
}

// Graph allows a user to manage graphs
type Graph struct {
	Operation `yaml:",inline"`
	// Smart indicates that the graph uses the Enterprise
	// edition's graph management.
	Smart *bool
	// SmartGraphAttribute is the attribute used to shuffle vertexes.
	SmartGraphAttribute string
	// Shards is the number of shards each collection has.
	Shards int
	// OrphanVertex
	OrphanVertices []string
	// EdgeDifinition creates a single edge between vertexes
	EdgeDefinitions []EdgeDefinition
	// Names of Edge Collections to remove
	RemoveEdges []string
	// Names of vertices to re
	RemoveVertices []string
}

// PairedMigrations Defines the primary change and an undo operation if provided.
// Presently undo is not a supported feature. After reading Flyway's
// history of the feature, it might  never be supported
type PairedMigrations struct {
	change Migration
	undo   Migration
}

// SearchView contains all the information needed to create an Arango Search SearchView.
type SearchView struct {
	Operation `yaml:",inline"`
	// CleanupIntervalStep specifies the minimum number of commits to wait between
	// removing unused files in the data directory.
	CleanupIntervalStep *int64 `yaml:"cleanupIntervalStep,omitempty"`
	// CommitInterval ArangoSearch waits at least this many milliseconds between committing
	// view data store changes and making documents visible to queries
	CommitIntervalMsec *int64 `yaml:"commitIntervalMsec,omitempty"`
	// ConsolidationInterval specifies the minimum number of milliseconds that must be waited
	// between committing index data changes and making them visible to queries.
	ConsolidationIntervalMsec *int64 `yaml:"consolidationIntervalMsec,omitempty"`
	// ConsolidationPolicy specifies thresholds for consolidation.
	ConsolidationPolicy *ConsolidationPolicy `yaml:"consolidationPolicy,omitempty"`
	// SortFields lists the fields that used for sorting.
	SortFields []SortField `yaml:"primarySort,omitempty"`
	// Links contains the properties for how individual collections
	// are indexed in thie view.
	Links []SearchElementProperties `yaml:"links,omitempty"`
}

// ConsolidationPolicy holds threshold values specifying when to
// consolidate view data.
// see ArangoSearchConsolidationPolicy
//     ArangoSearchConsolidationPolicyTier
//     ArangoSearchConsolidationPolicyBytesAccum
type ConsolidationPolicy struct {
	// Type returns the type of the ConsolidationPolicy.
	Type string
	// Options contains the fields used by the ConsolidationPolicy and are related to the Type.
	Options map[string]interface{}
}

// SortField describes a field and whether its ascending or not used for primary search.
type SortField struct {
	// The name of the field.
	Field string
	// Whether the field is ascending or descending.
	Ascending *bool `yaml:"ascending,omitempty"`
}

// SearchElementProperties contains properties that specify how an element
// is indexed in an ArangoSearch view.
// Note that this structure is recursive. Settings not specified (nil)
// at a given level will inherit their setting from a lower level.
type SearchElementProperties struct {
	// Name of the element (e.g. collection name)
	Name string
	// The list of analyzers to be used for indexing of string values. Defaults to ["identify"].
	// NOTE: They much be defined in Arango.
	Analyzers []string `yaml:"analyzers,omitempty"`
	// Fields contains the properties for individual fields of the element.
	Fields []SearchElementProperties `yaml:"fields,omitempty"`
	// If set to true, all fields of this element will be indexed. Defaults to false.
	IncludeAllFields *bool `yaml:"includeAllFields,omitempty"`
	// This values specifies how the view should track values.
	// see ArangoSearchStoreValues
	StoreValues *string `yaml:"storeValues,omitempty"`
	// If set to true, values in a listed are treated as separate values. Defaults to false.
	TrackListPositions *bool `yaml:"trackListPositions,omitempty"`
}

var validVersion = regexp.MustCompile(`^\d*(\.\d*)*?$`)

// Pairs migrations together.
// Returns an error if unable to find migrations.
func migrations(path string) ([]PairedMigrations, error) {
	migrations, err := loadFrom(path)
	if err != nil {
		return nil, err
	}
	if len(migrations) == 0 {
		return nil, errors.New("Could not find migrations at path '" + path + "'")
	}
	var pms []PairedMigrations

	for _, m := range migrations {
		pm := PairedMigrations{change: m, undo: nil}
		pms = append(pms, pm)
	}

	return pms, nil
}

func lpadToLength(s string, l int) string {
	dest := make([]rune, l)
	copy(dest[l-len(s):], []rune(s))
	return string(dest)
}

func version(s string) string {
	s = filepath.Base(s)
	idx := strings.IndexRune(s, '_')
	if idx == -1 {
		idx = strings.Index(s, ".migration")
	}
	out := s[:idx]
	if !validVersion.MatchString(out) {
		panic(fmt.Sprintf("File name doesn't match pattern: '%s'", s))
	}
	return out
}

// nearlyLexical sorts the paths based on near lexical sorting.
// Chomps the description of the migration off. Uses just the
// version information.
func nearlyLexical(s []string) func(i, j int) bool {
	return func(i, j int) bool {
		curV := version(s[i])
		toV := version(s[j])

		curVS := strings.Split(curV, ".")
		toVS := strings.Split(toV, ".")

		cL := len(curVS)
		tL := len(toVS)
		if cL < tL {
			t := make([]string, tL)
			copy(t, curVS)
			curVS = t
		} else if tL < cL {
			t := make([]string, cL)
			copy(t, toVS)
			toVS = t
		}

		for k, v := range curVS {
			to := toVS[k]
			vl := len(v)
			tl := len(to)
			if vl > tl {
				to = lpadToLength(to, vl)
			} else if vl < tl {
				v = lpadToLength(v, tl)
			}
			if v < to {
				return true
			} else if v > to {
				return false
			}
		}
		return false
	}
}

// Loads a set of migrations from a given directory.
func loadFrom(path string) ([]Migration, error) {
	parentDir := filepath.Join(path, "*.migration")
	migrations, err := filepath.Glob(parentDir)

	// This will destroy the whole process.
	if err != nil {
		return nil, err
	}

	// Attempts to sort by pseudo lexical means.
	sort.Slice(migrations, nearlyLexical(migrations))

	var answer []Migration
	for _, migration := range migrations {
		fmt.Printf("file name: %s\n", migration)
		as, err := toStruct(migration)
		if err != nil {
			return answer, err
		}
		fmt.Printf("The migration is %+v\n", as)
		answer = append(answer, as)
	}

	return answer, nil
}

// Opens the path into a byte slice.
// Returns the bytes, the file's checksum, and an error.
func open(childPath string) ([]byte, string, error) {
	bytes, err := ioutil.ReadFile(childPath)
	if err != nil {
		return nil, "", err
	}

	chk := md5.Sum(bytes)
	return bytes, hex.EncodeToString(chk[:]), nil
}

// Reads the migration contents to pick the proper type.
func pickT(contents []byte) (Migration, error) {
	s := string(contents)
	switch {
	case collection.MatchString(s):
		return new(Collection), nil
	case database.MatchString(s):
		return new(Database), nil
	case graph.MatchString(s):
		return new(Graph), nil
	case aql.MatchString(s):
		return new(AQL), nil
	case fulltextidx.MatchString(s):
		return new(FullTextIndex), nil
	case geoidx.MatchString(s):
		return new(GeoIndex), nil
	case hashidx.MatchString(s):
		return new(HashIndex), nil
	case persistentidx.MatchString(s):
		return new(PersistentIndex), nil
	case skipidx.MatchString(s):
		return new(SkiplistIndex), nil
	case view.MatchString(s):
		return new(SearchView), nil
	default:
		return nil, errors.New("Can't determine YAML type")
	}
}

/*
	Converts a path to the proper underlying types specified in
	the childPath.
*/
func toStruct(childPath string) (Migration, error) {
	contents, checksum, err := open(childPath)

	t, err := pickT(contents)
	if err != nil {
		return nil, err
	}

	err = yaml.UnmarshalStrict(contents, t)
	if err != nil {
		return nil, err
	}

	t.SetFileName(filepath.Base(childPath))
	t.SetCheckSum(checksum)
	return t, nil
}
