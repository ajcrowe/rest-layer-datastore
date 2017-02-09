package datastore

import (
	"context"
	"time"

	"cloud.google.com/go/datastore"
	"github.com/rs/rest-layer/resource"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// Wrap datastore.NewClient to avoid user having to import this
func NewClient(ctx context.Context, projectID string, opts ...option.ClientOption) (*datastore.Client, error) {
	return datastore.NewClient(ctx, projectID, opts...)
}

// Handler handles resource storage in Google Datastore.
type Handler struct {
	// datastore.Client struct for executing our queries.
	client *datastore.Client
	// Kind of the entity this handler will create.
	entity string
	// Namespace in which these entities will be.
	namespace string
	// Properties which should not be indexed.
	noIndexProps map[string]bool
}

// NewHandler creates a new Google Datastore handler
func NewHandler(client *datastore.Client, namespace, entity string) *Handler {
	return &Handler{
		client:    client,
		entity:    entity,
		namespace: namespace,
	}
}

// Entity Is a representation of a Google Datastore entity
type Entity struct {
	ID           string
	ETag         string
	Updated      time.Time
	Payload      map[string]interface{}
	NoIndexProps map[string]bool
}

// Load implements the PropertyLoadSaver interface to process our dynamic payload data
// see https://godoc.org/cloud.google.com/go/datastore#hdr-The_PropertyLoadSaver_Interface
func (e *Entity) Load(ps []datastore.Property) error {
	e.Payload = make(map[string]interface{}, len(ps)-3)
	for _, prop := range ps {
		// Load our hard coded fields if property name matches
		// otherwise load the dynamic property into Payload map
		switch prop.Name {
		case "_id":
			e.ID = prop.Value.(string)
		case "_etag":
			e.ETag = prop.Value.(string)
		case "_updated":
			e.Updated = prop.Value.(time.Time)
		default:
			e.Payload[prop.Name] = prop.Value
		}
	}
	return nil
}

// Save implements the PropertyLoadSaver interface to process our dynamic payload data
// see https://godoc.org/cloud.google.com/go/datastore#hdr-The_PropertyLoadSaver_Interface
func (e *Entity) Save() ([]datastore.Property, error) {
	// Create our default struct properties
	ps := []datastore.Property{
		datastore.Property{
			Name:  "_id",
			Value: e.ID,
		},
		datastore.Property{
			Name:  "_etag",
			Value: e.ETag,
		},
		datastore.Property{
			Name:  "_updated",
			Value: e.Updated,
		},
	}
	// Range over the payload and create the datastore.Properties
	for k, v := range e.Payload {
		prop := datastore.Property{
			Name:    k,
			Value:   v,
			NoIndex: e.NoIndexProps[k],
		}
		ps = append(ps, prop)
	}
	return ps, nil
}

// newItem converts datastore entity into a resource.Item
func newItem(e *Entity) *resource.Item {
	e.Payload["id"] = e.ID
	return &resource.Item{
		ID:      e.ID,
		ETag:    e.ETag,
		Updated: e.Updated,
		Payload: e.Payload,
	}
}

// newEntity converts a resource.Item into a Google datastore entity
func (d *Handler) newEntity(i *resource.Item) *Entity {
	p := make(map[string]interface{}, len(i.Payload))
	for k, v := range i.Payload {
		if k != "id" {
			p[k] = v
		}
	}
	return &Entity{
		ID:           i.ID.(string),
		ETag:         i.ETag,
		Updated:      i.Updated,
		Payload:      p,
		NoIndexProps: d.noIndexProps,
	}
}

// SetNoIndexProps sets the handlers properties which should have noindex set.
func (d *Handler) SetNoIndexProperties(props []string) *Handler {
	p := make(map[string]bool, len(props))
	for _, v := range props {
		p[v] = true
	}
	d.noIndexProps = p
	return d
}

// Insert inserts new entities
func (d *Handler) Insert(ctx context.Context, items []*resource.Item) error {
	mKeys := make([]*datastore.Key, len(items))
	mEntities := make([]interface{}, len(items))

	for i, item := range items {
		mKeys[i] = datastore.NameKey(d.entity, item.ID.(string), nil)
		mKeys[i].Namespace = d.namespace
		mEntities[i] = d.newEntity(item)
	}
	_, err := d.client.PutMulti(ctx, mKeys, mEntities)
	return err
}

// Update replace an entity by a new one in the Datastore
func (d *Handler) Update(ctx context.Context, item *resource.Item, original *resource.Item) error {
	var err error

	entity := d.newEntity(item)
	// Run a transaction to update the Entity if the Entity exist and the ETags match
	tx := func(tx *datastore.Transaction) error {
		// Create a key for our current Entity
		key := datastore.NameKey(d.entity, original.ID.(string), nil)
		key.Namespace = d.namespace

		var current Entity
		// Attempt to get the existing Entity
		if err = tx.Get(key, &current); err != nil {
			if err == datastore.ErrNoSuchEntity {
				return resource.ErrNotFound
			}
			return err
		}
		if current.ETag != original.ETag {
			return resource.ErrConflict
		}
		// Update the Entity
		_, err = tx.Put(key, entity)
		return err
	}
	_, err = d.client.RunInTransaction(ctx, tx, datastore.MaxAttempts(1))
	return err
}

// Delete deletes an item from the datastore
func (d *Handler) Delete(ctx context.Context, item *resource.Item) error {
	var err error
	// Run a transaction to update the Entity if the Entity exist and the ETags match
	tx := func(tx *datastore.Transaction) error {
		// Create a key for our target Entity
		key := datastore.NameKey(d.entity, item.ID.(string), nil)
		key.Namespace = d.namespace

		var e Entity
		// Attempt to get the existing Entity
		if err = tx.Get(key, &e); err != nil {
			if err == datastore.ErrNoSuchEntity {
				return resource.ErrNotFound
			}
			return err
		}
		if e.ETag != item.ETag {
			return resource.ErrConflict
		}
		// Delete the Entity
		err = tx.Delete(key)
		return err
	}
	_, err = d.client.RunInTransaction(ctx, tx, datastore.MaxAttempts(1))
	return err
}

// Clear clears all entities matching the lookup from the Datastore
func (d *Handler) Clear(ctx context.Context, lookup *resource.Lookup) (int, error) {
	q, err := getQuery(d.entity, d.namespace, lookup)
	if err != nil {
		return 0, err
	}

	c, err := d.client.Count(ctx, q)
	if err != nil {
		return 0, err
	}

	// TODO: Check wheter if DeleteMulti is better here than delete on every
	// iteration here or not.
	mKeys := make([]*datastore.Key, c)
	for t, i := d.client.Run(ctx, q), 0; ; i++ {
		var e Entity
		key, err := t.Next(&e)
		mKeys[i] = key
		if err == iterator.Done {
			break
		}
	}

	err = d.client.DeleteMulti(ctx, mKeys)
	if err != nil {
		return 0, err
	}
	return len(mKeys), nil
}

// Find entities matching the provided lookup from the Datastore
func (d *Handler) Find(ctx context.Context, lookup *resource.Lookup, offset, limit int) (*resource.ItemList, error) {
	q, err := getQuery(d.entity, d.namespace, lookup)
	if err != nil {
		return nil, err
	}

	// TODO: Apply context deadline if any.
	list := &resource.ItemList{Total: -1, Offset: offset, Limit: limit, Items: []*resource.Item{}}
	for t := d.client.Run(ctx, q); ; {
		var e Entity
		_, terr := t.Next(&e)
		if terr == iterator.Done {
			break
		}
		if terr != nil {
			return nil, terr
		}
		if terr = ctx.Err(); terr != nil {
			return nil, terr
		}
		list.Items = append(list.Items, newItem(&e))
	}
	return list, nil
}
