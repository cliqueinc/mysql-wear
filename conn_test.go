package mwear

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	TestPort     = 3406
	TestUser     = "root"
	TestPassword = "root"
)

var db *DB

/*
Configure the running of the main to create and config a temp/test db
*/
func TestMain(m *testing.M) {
	flag.Parse()
	// envDBName := os.Getenv("MYSQL_DB")
	envDBName := "mw_tmp"
	// debugEnabled = true

	var shouldDropDB bool
	if !strings.HasPrefix(envDBName, "mw_tmp") {
		// Create and new a tmp db
		envDBName = CreateDB("mw_tmp")
		shouldDropDB = true

	}

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
