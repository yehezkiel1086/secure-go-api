package migrations

import (
	_ "embed"
)

// Version identifies the initial schema migration.
const Version = "20260904_initial_schema"

//go:embed schema.sql
var SchemaSQL string
