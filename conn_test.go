package mwear

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"testing"
	"time"
)

const (
	TestPort     = 3306
	TestUser     = "root"
	TestPassword = "testpass1234"
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
		envDBName = CreateDB("mw_tmp")
		shouldDropDB = true

	}

	// TODO combine with core way of connecting to mysql so we don't forget to change this in two places
	dsn := fmt.Sprintf(
		"%s:%s@tcp(127.0.0.1:%d)/%s?tls=false&parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci&sql_mode=''",
		TestUser,
		TestPassword,
		TestPort,
		envDBName,
	)
	mysqlDB, err := sql.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}
	db = New(mysqlDB)

	exitVal := m.Run()
	if shouldDropDB {
		DropDB(envDBName)
	}
	mysqlDB.Close()
	os.Exit(exitVal)
}

/*
These should only be used during local testing
*/
func CreateDB(namePrefix string) string {
	t := time.Now()
	dbName := fmt.Sprintf("%s_%s", namePrefix, t.Format("2006-01-02-15:04:05"))
	// Create the tmp db, make sure can connect to its
	out, err := exec.Command(
		"mysqladmin",
		"-h127.0.0.1",
		"-P"+strconv.Itoa(TestPort),
		"-u"+TestUser,
		"-p"+TestPassword,
		"create",
		dbName,
	).CombinedOutput()

	if err != nil {
		panic(fmt.Sprintf("Failed to created temporary DB, Output: (%s) err (%s)\n", out, err))
	}

	fmt.Println("Created new tmp testing db", dbName)
	return dbName
}

func DropDB(dbName string) {
	out, err := exec.Command(
		"mysqladmin",
		"-f", // Need the -f to force drop db otherwise will prompt y/n
		"-h127.0.0.1",
		"-P"+strconv.Itoa(TestPort),
		"-u"+TestUser,
		"-p"+TestPassword,
		"drop",
		dbName,
	).CombinedOutput()

	if err != nil {
		panic(fmt.Sprintf("Failed to drop temporary DB, Output: (%s) err (%s)\n", out, err))
	}
	fmt.Println("Dumped testing db", dbName)
}
