package mwear

import (
	"database/sql"
	"fmt"
	"io"
)

var debugEnabled bool

// DB is a wrapper aroung standard sql db that also wraps common sql opperations.
type DB struct {
	DB     *sql.DB
	Logger io.Writer
	*Adapter
}

// New initializes new mysql-wear client assuming that sql connection already has been configured.
func New(db *sql.DB) *DB {
	return &DB{db, nil, Wrap(db)}
}

func (db *DB) Log(msg string, args ...interface{}) {
	if db.Logger != nil {
		db.Logger.Write([]byte(fmt.Sprintf(msg, args...) + "\n"))
	}
}

// Begin starts the transaction.
func (db *DB) Begin() (*sql.Tx, error) {
	return db.DB.Begin()
}
