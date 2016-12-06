package datastore_test

import (
	"log"
	"net/http"

	gds "cloud.google.com/go/datastore"
	"github.com/ajcrowe/rest-layer-datastore"
	"github.com/rs/cors"
	"github.com/rs/rest-layer/resource"
	"github.com/rs/rest-layer/rest"
	"github.com/rs/rest-layer/schema"
	"google.golang.org/appengine/aetest"
)

var (
	user = schema.Schema{
		Fields: schema.Fields{
			"id":      schema.IDField,
			"created": schema.CreatedField,
			"updated": schema.UpdatedField,
			"name": {
				Required:   true,
				Filterable: true,
				Sortable:   true,
				Validator: &schema.String{
					MaxLen: 150,
				},
			},
		},
	}

	// Define a post resource schema
	post = schema.Schema{
		Fields: schema.Fields{
			"id":      schema.IDField,
			"created": schema.CreatedField,
			"updated": schema.UpdatedField,
			"user": {
				Required:   true,
				Filterable: true,
				Validator: &schema.Reference{
					Path: "users",
				},
			},
			"public": {
				Filterable: true,
				Validator:  &schema.Bool{},
			},
			"meta": {
				Schema: &schema.Schema{
					Fields: schema.Fields{
						"title": {
							Required: true,
							Validator: &schema.String{
								MaxLen: 150,
							},
						},
						"body": {
							Validator: &schema.String{
								MaxLen: 100000,
							},
						},
					},
				},
			},
		},
	}
)

func Example() {
	ctx, done, err := aetest.NewContext()
	if err != nil {
		log.Fatal(err)
	}
	defer done()

	client, _ := gds.NewClient(ctx, "test-project")

	index := resource.NewIndex()

	users := index.Bind("users", user, datastore.NewHandler(client, "default", "users"), resource.Conf{
		AllowedModes: resource.ReadWrite,
	})

	users.Bind("posts", "user", post, datastore.NewHandler(client, "default", "posts"), resource.Conf{
		AllowedModes: resource.ReadWrite,
	})

	api, err := rest.NewHandler(index)
	if err != nil {
		log.Fatalf("Invalid API configuration: %s", err)
	}

	http.Handle("/", cors.New(cors.Options{OptionsPassthrough: true}).Handler(api))

	log.Print("Serving API on http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
