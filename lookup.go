package datastore

import (
	"fmt"
	"reflect"

	"cloud.google.com/go/datastore"
	"github.com/rs/rest-layer/resource"
	"github.com/rs/rest-layer/schema/query"
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
func getQuery(e string, ns string, q *query.Query) (*datastore.Query, error) {
	query, err := translateQuery(datastore.NewQuery(e), q.Predicate)
	// if lookup specifies sorting add this to our query
	if len(q.Sort) > 0 {
		s := make([]string, len(q.Sort))
		for i, sort := range q.Sort {
			if sort.Reversed {
				s[i] = "-" + getField(sort.Name)
			} else {
				s[i] = getField(sort.Name)
			}
		}
		query = getSort(query, s)
	}
	// Set namespace for this query
	query = query.Namespace(ns)
	return query, err
}

func translateQuery(dsQuery *datastore.Query, q query.Predicate) (*datastore.Query, error) {
	var err error
	// process each schema.Expression into a datastore filter
	for _, exp := range q {
		switch t := exp.(type) {
		case *query.Equal:
			// If our Query contains a slice, add each as an additional filter
			if reflect.TypeOf(t.Value).Kind() == reflect.Slice {
				for _, v := range t.Value.([]interface{}) {
					dsQuery = dsQuery.Filter(fmt.Sprintf("%s =", getField(t.Field)), v)
				}
			} else {
				dsQuery = dsQuery.Filter(fmt.Sprintf("%s =", getField(t.Field)), t.Value)
			}
		case *query.NotEqual:
			dsQuery = dsQuery.Filter(fmt.Sprintf("%s !=", getField(t.Field)), t.Value)
		case *query.GreaterThan:
			dsQuery = dsQuery.Filter(fmt.Sprintf("%s >", getField(t.Field)), t.Value)
		case *query.GreaterOrEqual:
			dsQuery = dsQuery.Filter(fmt.Sprintf("%s >=", getField(t.Field)), t.Value)
		case *query.LowerThan:
			dsQuery = dsQuery.Filter(fmt.Sprintf("%s <", getField(t.Field)), t.Value)
		case *query.LowerOrEqual:
			dsQuery = dsQuery.Filter(fmt.Sprintf("%s <=", getField(t.Field)), t.Value)
		case *query.And:
			for _, subExp := range *t {
				dsQuery, err = translateQuery(dsQuery, query.Predicate{subExp})
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
