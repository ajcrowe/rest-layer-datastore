package datastore

import (
	"fmt"

	//"github.com/davecgh/go-spew/spew"

	"cloud.google.com/go/datastore"
	"github.com/rs/rest-layer/resource"
	"github.com/rs/rest-layer/schema"
)

// getSort transform a resource.Lookup into a Datastore sort list.
// If the sort list is empty, fallback to _id.
func getSort(query *datastore.Query, s []string) *datastore.Query {

	for _, sort := range s {
		query = query.Order(sort)
	}
	return query
}

// getField translates id to _id to avoid duplication
func getField(f string) string {
	if f == "id" {
		return "_id"
	}
	return f
}

// getQuery transform a resource.Lookup into a Google Datastore query
func getQuery(e string, l *resource.Lookup) (*datastore.Query, error) {
	query, err := translateQuery(datastore.NewQuery(e), l.Filter())

	// if lookup specifies sorting add this to our query
	if len(l.Sort()) > 0 {
		query = getSort(query, l.Sort())
	}
	return query, err
}

func translateQuery(dsQuery *datastore.Query, q schema.Query) (*datastore.Query, error) {
	var err error
	// process each schema.Expression into a datastore filter
	for _, exp := range q {
		switch t := exp.(type) {
		case schema.Equal:
			dsQuery = dsQuery.Filter(fmt.Sprintf("%s =", getField(t.Field)), t.Value)
		case schema.NotEqual:
			dsQuery = dsQuery.Filter(fmt.Sprintf("%s !=", getField(t.Field)), t.Value)
		case schema.GreaterThan:
			dsQuery = dsQuery.Filter(fmt.Sprintf("%s >", getField(t.Field)), t.Value)
		case schema.GreaterOrEqual:
			dsQuery = dsQuery.Filter(fmt.Sprintf("%s >=", getField(t.Field)), t.Value)
		case schema.LowerThan:
			dsQuery = dsQuery.Filter(fmt.Sprintf("%s <", getField(t.Field)), t.Value)
		case schema.LowerOrEqual:
			dsQuery = dsQuery.Filter(fmt.Sprintf("%s <=", getField(t.Field)), t.Value)
		case schema.And:
			for _, subExp := range t {
				dsQuery, err = translateQuery(dsQuery, schema.Query{subExp})
				if err != nil {
					return nil, err
				}
			}
		default:
			// return resource.ErrNotImplemented for:
			// schema.Or, schema.In, schema,NotIn
			return nil, resource.ErrNotImplemented
		}
	}
	return dsQuery, nil
}
