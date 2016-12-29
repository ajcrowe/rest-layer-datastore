# REST Layer Google Datastore Backend

[![godoc](http://img.shields.io/badge/godoc-reference-blue.svg?style=flat)](https://godoc.org/github.com/ajcrowe/rest-layer-datastore) [![license](http://img.shields.io/badge/license-MIT-red.svg?style=flat)](https://raw.githubusercontent.com/ajcrowe/rest-layer-datastore/master/LICENSE) [![build](https://img.shields.io/travis/ajcrowe/rest-layer-datastore.svg?style=flat)](https://travis-ci.org/ajcrowe/rest-layer-datastore)

This [REST Layer](https://github.com/rs/rest-layer) resource storage backend stores data in a Google Datastore using [datastore](https://godoc.org/cloud.google.com/go/datastore).

This backend used [cmorent/rest-layer-datastore](https://github.com/cmorent/rest-layer-datastore) as a base and borrows structure and approach from [rs/rest-layer-mongo](https://github.com/rs/rest-layer-mongo). It uses the [general](https://godoc.org/cloud.google.com/go/datastore) library rather than specific [appengine](https://google.golang.org/appengine/datastore) library.

## Usage

```go
import "github.com/ajcrowe/rest-layer-datastore"
```

Create a datastore client

```go
ctx := context.Background()
client, err := datastore.NewClient(ctx, "project-id")
if err != nil {
	log.Fatalf("Error connecting to Datastore: %s", err)
}
```

Then use this to create a new `Handler` for your resource binds

```go
// params for the handler
namespace := "default"
entity := "users"
// bind the users resource with the datastore handler
index.Bind("users", user, datastore.NewHandler(client, namespace, entity), resource.DefaultConf)
```

You can also set a number of Datastore properties which you would like to exclude from being indexed with `SetNoIndexProperties` on your `handler` struct.

```go
// create a handler for the resource.
index.Bind("users", user, datastore.NewHandler(client, namespace, entity).SetNoIndexProperties([]string{"prop1", "prop2"}), resource.DefaultConf)

```

## Supported filter operators

- [x] $and
- [ ] $or
- [x] $lt
- [x] $lte
- [x] $gt
- [x] $gte
- [ ] $in
- [ ] $nin
- [ ] $exists

