package mwear

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"testing"
	"time"
)

var db *DB

/*
Configure the running of the main to create and config a temp/test db
*/
func TestMain(m *testing.M) {
	flag.Parse() // Why? I know there's a reason...

	cv, err := ReadEnvFile(".mysql.env")
	if err != nil {
		log.Printf("Error: %s", err)
		log.Println("Make sure .env.mysql exists")
		os.Exit(-1)
	}
	mysqlDB, err := sql.Open("mysql", cv.ConnectString())
	if err != nil {
		panic(err)
	}
	tmpDBPrefix := "mw_tmp"
	tmpDBName := CreateDB(mysqlDB, tmpDBPrefix)

	if _, err := mysqlDB.Exec("USE " + tmpDBName); err != nil {
		panic(fmt.Sprintf("Failed to switch DB, error (%s)\n", err))
	}

	db = New(mysqlDB)

	exitVal := m.Run()
	DropDB(mysqlDB, tmpDBName)
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
