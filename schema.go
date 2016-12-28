package datastore

import (
	"github.com/rs/rest-layer/schema"
)

type Fields map[string]Field

// Wrap common schema fields from rest-layer
var (
	IDField = Field{
		Field: schema.IDField,
	}

	CreatedField = Field{
		Field:   schema.CreatedField,
		NoIndex: true,
	}

	UpdatedField = Field{
		Field:   schema.UpdatedField,
		NoIndex: true,
	}

	PasswordField = Field{
		Field:   schema.PasswordField,
		NoIndex: true,
	}
)

type Field struct {
	schema.Field
	NoIndex bool
}
