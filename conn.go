package mwear

import (
	"database/sql"
)

var debugEnabled bool

// DB is a wrapper aroung standard sql db that also wraps common sql opperations.
type DB struct {
	DB *sql.DB
	*Adapter
}

// New initializes new mysql-wear client assuming that sql connection already has been configured.
// Use this when you want to manually configure your mysql connection string
// (not recommended)
func New(db *sql.DB) *DB {
	return &DB{db, Wrap(db)}
}

// Begin starts the transaction.
func (db *DB) Begin() (*sql.Tx, error) {
	return db.DB.Begin()
}

// Init mysql and mwear
func Init() {}

// InitFromEnvVars Loads everything up from os.Env vars
func InitFromEnvVars() {}
