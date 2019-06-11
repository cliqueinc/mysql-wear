package mwear_test

import (
	"database/sql"
	"fmt"
	"time"

	mw "github.com/cliqueinc/mysql-wear"
	"github.com/cliqueinc/mysql-wear/sqlq"
)

func ExampleSelect() {
	type user struct {
		Name string
	}

	// prepare database connection
	dsn := fmt.Sprintf(
		"%s:%s@tcp(127.0.0.1:%d)/%s?tls=false&parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci&sql_mode=''",
		"root",
		"password",
		3606,
		"db_name",
	)
	mysqlDB, err := sql.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}
	db := mw.New(mysqlDB)

	var users []user
	db.Select(
		&users,
		sqlq.Equal("id", 123),
		sqlq.OR(
			sqlq.Equal("id", 123),
			sqlq.LessThan("name", 123),
			sqlq.AND(
				sqlq.NotEqual("id", 123),
				sqlq.GreaterOrEqual("name", 123),
			),
		),
		sqlq.Limit(10),
		sqlq.Order("id", sqlq.DESC),
	)

	type fakeBlog struct {
		Name         string
		Descr        string
		ID           string
		PublishStart time.Time
	}

	var fetchedBlogs []fakeBlog
	db.Select(
		&fetchedBlogs,
		sqlq.OR(
			sqlq.Equal("name", "blog4"),
			sqlq.AND(
				sqlq.Equal("descr", "descr3"),
				sqlq.IN("id", "111", "222"),
			),
		),
	)
}
