package mwear

import (
	"database/sql"
	"fmt"
)

// TODO is this being used? (grep)
var debugEnabled bool

// DB is a wrapper around standard sql db that also wraps common sql opperations.
type DB struct {
	DB *sql.DB
	*Adapter
}

// ConnectVals hold connection info for mysql
// UnixSocket should be a full path like for CloudSQL: /cloudsql/PROJECT_ID:REGION_ID:INSTANCE_ID
type ConnectVals struct {
	Host       string
	UnixSocket string // Supersedes tcp connection
	Port       int
	DBName     string
	UserName   string
	Password   string

	// Required for InitWithSchema
	MigrationPath string
	UseTLS        bool // Should default to true
}

func (cv ConnectVals) TLS() string {
	if cv.UseTLS {
		return "true"
	}
	return "false"
}

// TODO test me
func (cv ConnectVals) ConnectString() string {
	var connStr string
	if cv.UnixSocket != "" { // Unix Socket takes precedence over tcp/port
		connStr = fmt.Sprintf(
			"%s:%s@unix(%s)/",
			cv.UserName,
			cv.Password,
			cv.UnixSocket,
		)
	} else {
		connStr = fmt.Sprintf(
			"%s:%s@tcp(%s:%d)/",
			cv.UserName,
			cv.Password,
			cv.Host,
			cv.Port,
		)
	}
	if cv.DBName != "" { // Only add the DB name if set
		connStr += cv.DBName
	}
	connStr += fmt.Sprintf(
		"?multiStatements=true&tls=%s&parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci&sql_mode=''",
		cv.TLS(),
	)
	return connStr
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

// Init mysql, mw, load and update any schema
func InitWithSchema(cv ConnectVals) (*DB, error) {
	// TODO we probably don't want to use multi statements after schema init
	dsn := cv.ConnectString()
	mysqlDB, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	err = RegisterMigrationPath(cv.MigrationPath)
	if err != nil {
		return nil, err
	}

	mwDB := New(mysqlDB)
	if err := mwDB.InitSchema(false); err != nil {
		return nil, err
	}
	// Runs any schema updates
	if err := mwDB.UpdateSchema(false); err != nil {
		return nil, err
	}
	return mwDB, nil
}

// InitFromEnvVars Loads everything up from os.Env vars
func InitFromEnvVars() {}
