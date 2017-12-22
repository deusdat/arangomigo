# ArangoMiGO

A schema evolution tool for [ArangoDB](https://arangodb.com/). Manage your collections, indices, 
and data transformations in a centralized way in your source control along with your application.

The goal behind the project is to apply to ArangoDB years of hard fought lessons (especially those that kicked us in the teeth). We needed a schema version manager that could create a database, add all of the collections, indexes and data population necessary for a developer to create a local VM 
of ArangoDB that looks like a mini-production. The system should automatically adjust to merges. While providing all of this, it must also support the neat features we all know and chose ArangoDB for: 
sharding collections on distributed systems. This means that we can't rely on creating the collection automatically if it doesn't exist when inserting a document. Sometimes collections should come preloaded with some documents from start.

### Supports Arango 3.1+.

## Getting an executable
If you're familiar with Go, you can clone and build this project directly on your target machine. If you'd prefer an official build, look in the dist directory of the project.

To your executable pass the path to the configuration file, which is defined below. 

## Creating your structures

ArangoMiGO supports creating, modifying, and deleting graphs, collections, indexes, and even the database. Below you'll see how to use YAML to create a migration set. Once a migration component executes, the system doesn't rerun it. You don't have to worry about creating a collection or running data migration twice.

### Creating the configuration file
```yaml
endpoints:
   - http://arangodb-local:8529
username: root
password: devroot
migrationspath: /home/jdavenpo/go/src/github.com/deusdat/arangomigo/testdata/complete
db: MigoFull
extras:
  {patricksUser: jdavenpo,
   patricksPassword: Extrem!Password&^%$#,
   shouldBeANumber: 10,
   secret: Lots of mayo}

```
ArangoMiGO supports fail over out of the box: `endpoints`. If you are creating a database as part of the
migration set, make sure that the username has the proper rights. 

`migrationspath` is the directory holding
the migration configurations. At this time ArangoMiGO does not support nested directories. 

`db` is the name of the target database. If you create the database as part of the migration, the name in the config and in the migration must match.

`extras` allows you to specify arbitrary values through a look up mechanism. As you'll see later, you can use ${} to mark fields, such as those found in the BindVars of the AQL migration, as replaceable. This allows you to add sensitive data that should not go in source control.

Did we mention that you shouldn't store the config in source control? No? Don't store the config in source control.

### A quick note on versioning
Each step in the migration set is another version. If you're familiar with liquibase, you give the change a specific id. Flyway uses the file name format, as does ArangoMiGO. 

File names have this pattern: VersionNumber<_Any_description>.migration. A version number has to be in the format of number.<number>. Here are a few examples.
  * 1.migration
  * 1.4_Adds_ROI_CALC_FUNCTION.migration
  * 12.6.7.2.migration
  * 19.02.02.migration

Let's say you start with the following migration set.
  * 1.migration
  * 2.migration
  * 3.migration
  * 4.migration

Then you need to add an index for a collection created in `3.migration`. You can either create `5.migration` or `3.1.migration`. ArangoMiGO will see that it's applied 3, but not 3.1 and apply it. Either way works. The latter is more logically consisent for a new deploy.

ArangoMiGO halts at the first failure. Other systems solider through error and report them at the end. In our experience this is a bad idea when it comes to our data. We baked that philosophy in.

### Creating your database
```yaml
type: database
action: create
name: MigoFull
allowed:
  - username: ${patricksUser}
    password: ${patricksPassword}
```

One thing to notice is that the user name and password leverage the replacement feature. You can safely commit this migration without fear of tipping your security hand in the future.

You can also include a list of users not allowed in the database with `Disallowed` field.

### Dropping your database
```yaml
type: database
action: delete
name: MigoFull
```
Deletes a database with the name of MigoFull

### Creating a collection
```yaml 
type: collection
action: create
name: recipes
journalsize: 10485760
waitforsync: true
```
This example creates a collection in the database named in the config file and sets the journalsize and wait for sync properties. You can also add the following features.
  * shardkeys - list of fields to use for the shard key.
  * numberofshards - integer
  * allowuserkeys - boolean 
  * volatile - boolean
  * compactable - boolean
  * waitforsync - boolean
  * journalsize - int

If you don't include a specific property, Arango applies its own default.

### Modifying a collection
```yaml 
type: collection
action: modify
name: recipes
journalsize: 10485760
waitforsync: true
```
You can exclude either `journalsize` or `waitforsync`.

### Deleting a collection
```yaml 
type: collection
action: delete
name: recipes
```

### Creating a graph

```yaml
type: graph
action: create
name: testing_graph
edgedefinitions:
   - collection: relationships
     from: 
         - recipes
     to: 
         - recipes
```

Creates a graph named `testing_graph` with one edge between the collection vertex recipes named relationships. You can also set the following attributes.
  * smart - bool if you are using the Enterprise edition. 
  * smartgraphattribute - string the attribute used to shuffle vertexes.
	* shards - int the number of shards each collection has.
	* orphanvertices - []string a list of collections within the graph, but not part of an edge.
	* edgedefinitions - []EdgeDefinition creates a single edge between vertexes, where EdgeDefinition looks like the on in the example above.

### Modify a graph
This example modifies the graph `testing_graph` by adding a new edge `owns` and changing the existing edge `relationship` to include users as a target. Finally, this adds a vertex `another` to the orphan vertices collections.
```yaml
type: graph
action: modify
name: testing_graph
edgedefinitions:
   - collection: owns
     from: 
         - users
     to: 
         - recipes
   - collection: relationships
     from: 
         - recipes
     to: 
         - recipes
         - users
orphanvertices:
   - another
```
It is possible that a graph could be partially configured. If you specified a series of changes like removing orphan vertices and adding new edges, that the vertices maybe deleted, but the edges won't be added. Please watch the output for warnings.

You must specify the graph, action as modify and the name of the graph. You can use these attributes to make changes.
  * removevertices - []string names of the vertices to remove. If you attempt to remove a vertex included in an edge, the migration will fail.
  * removeedges - []string the names of the edges you want to remove.
  * orphanvertices - []string allows you to add vertices to the graph without included them in the edges. It will create a new vertex if the collection does not already exist.
  * edgedefinitions - []EdgeDefinition names an edge and vertices that comprise the To and From. If the edge definition already exists, it gets updated to reflect the To, From relationship.

### Delete a graph
```yaml
type: graph
action: delete
name: testing_graph
```

### Indexes
At present you can only create indexes. ArangoDB doesn't expose an API to properly identify indexes.
