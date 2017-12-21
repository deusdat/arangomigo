# ArangoMiGO

A schema evolution tool for [ArangoDB](https://arangodb.com/). Manage your collections, indices, 
and data transformations in a centralized way in your source control along with your application.

The goal behind the project is to apply to ArangoDB years of hard fought lessons (especially those that kicked us in the teeth). We needed a schema version manager that could create a database, add all of the collections, indexes and data population necessary for a developer to create a local VM 
of ArangoDB that looks like a mini-production. The system should automatically adjust to merges. While providing all of this, it must also support the neat features we all know and chose ArangoDB for: 
sharding collections on distributed systems. This means that we can't rely on creating the collection automatically if it doesn't exist when inserting a document. Sometimes collections should come preloaded with some documents from start.

Supports Arango 3.1+.

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
   patricksPassword: L33t5uck3r5,
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
  * 1
  * 1.4
  * 12.6.7.2

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