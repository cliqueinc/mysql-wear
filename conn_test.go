package mwear

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"testing"
	"time"
)

const (
	TestHost     = "127.0.0.1"
	TestPort     = 3306
	TestUser     = "root"         // root user
	TestPassword = "testpass1234" // root password
)

var db *DB

/*
Configure the running of the main to create and config a temp/test db
*/
func TestMain(m *testing.M) {
	flag.Parse()
	// envDBName := os.Getenv("MYSQL_DB")
	envDBName := "mw_tmp" // See **Note** below. CreateDB will set this
	// debugEnabled = true

	// TODO combine with core way of connecting to mysql so we don't forget to change this in two places
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/?tls=false&parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci&sql_mode=''",
		TestUser,
		TestPassword,
		TestHost,
		TestPort,
	)
	mysqlDB, err := sql.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}

	var shouldDropDB bool
	// **Note** This logic pre-dates the current version of mysql-wear!
	// What this is doing is making sure your configured dbname (which is SUPPOSED
	// to come from an env variable) isn't something real like myworkproject.
	// SO if it DOESNT start with mw_tmp, we override and give you a tmp DB then drop it.
	// For now just going to comment that check since we always want to use a tmp db
	//
	// if !strings.HasPrefix(envDBName, "mw_tmp") {
	if true {
		// Create and new a tmp db
		envDBName = CreateDB(mysqlDB, "mw_tmp")
		shouldDropDB = true

	}

	if _, err := mysqlDB.Exec("USE " + envDBName); err != nil {
		panic(fmt.Sprintf("Failed to switch DB, error (%s)\n", err))
	}

	db = New(mysqlDB)

	exitVal := m.Run()
	if shouldDropDB {
		DropDB(mysqlDB, envDBName)
	}
	mysqlDB.Close()
	os.Exit(exitVal)
}

/*
These should only be used during local testing
*/
func CreateDB(mysqlDB *sql.DB, namePrefix string) string {
	t := time.Now()
	dbName := fmt.Sprintf("%s_%s", namePrefix, t.Format("2006_01_02_15_04_05"))

	_, err := mysqlDB.Exec("CREATE DATABASE " + dbName)
	if err != nil {
		panic(fmt.Sprintf("Failed to created temporary DB, error (%s)\n", err))
	}

	fmt.Println("Created new tmp testing db", dbName)
	return dbName
}

func DropDB(mysqlDB *sql.DB, dbName string) {
	_, err := mysqlDB.Exec("DROP DATABASE " + dbName)

	if err != nil {
		panic(fmt.Sprintf("Failed to drop temporary DB, error (%s)\n", err))
	}
	fmt.Println("Dumped testing db", dbName)
}
