package dbal

import "github.com/jmoiron/sqlx"

// Grammar the SQL Grammar inteface
type Grammar interface {
	Config(dsn string)

	GetVersion(db *sqlx.DB) (*Version, error)
	GetDBName() string
	GetSchemaName() string

	Exists(name string, db *sqlx.DB) bool
	Get(table *Table, db *sqlx.DB) error
	Create(table *Table, db *sqlx.DB) error
	Alter(table *Table, db *sqlx.DB) error
	Drop(name string, db *sqlx.DB) error
	DropIfExists(name string, db *sqlx.DB) error
	Rename(old string, new string, db *sqlx.DB) error
	GetColumnListing(dbName string, tableName string, db *sqlx.DB) ([]*Column, error)
}

// Quoter the database quoting query text intrface
type Quoter interface {
	ID(name string, db *sqlx.DB) string
	VAL(v interface{}, db *sqlx.DB) string // operates on both string and []byte and int or other types.
}