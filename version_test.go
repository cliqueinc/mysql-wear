package mwear

import (
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
)

func TestMigrationHandler(t *testing.T) {
	type User struct {
		ID      string
		Name    string
		Created time.Time
	}

	migrationHandler := &MigrationHandler{}
	migrationHandler.RegisterMigration(
		"2017-09-15:15:08:52",
		GenerateSchema(&User{}),
		`DROP TABLE user;`,
	)
	migrationHandler.RegisterMigration(
		"2017-09-16:15:08:52",
		`insert into user (id, name, created) VALUES 
		('111','user1','2006-01-02:15:04:05'), 
		('222','user2','2006-01-02:15:04:05');`,
		`delete from user where name IN ('user1', 'user2')`,
	)
	migrationHandler.RegisterMigration(
		DefaultVersion,
		"insert into `user` (id, name, created) VALUES ('333','user3','2006-01-02:15:04:05');",
		"delete from `user` where name = 'user3'",
	)
	mh = migrationHandler
	if err := db.InitSchema(false); err != nil {
		t.Fatalf("fail init schema: %v", err)
	}
	if err := db.UpdateSchema(false); err != nil {
		t.Fatalf("fail update schema: %v", err)
	}

	// check all migrations were applied
	count, err := db.Count(&SchemaMigration{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 2 {
		t.Fatalf("not all migration were applied: expected (%d), actual applied (%d)", 2, count)
	}

	// check that only 1st and 2nd migration were executed
	count, err = db.Count(&User{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected %d users in table, actual (%d)", 2, count)
	}

	if err := db.Rollback("2017-09-16:15:08:52"); err != nil {
		t.Fatalf("fail rollback 2nd migration: %v", err)
	}
	count, err = db.Count(&User{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 {
		t.Fatalf("2nd migration didn't roll back")
	}

	if err := db.ExecuteMigration("2017-09-16:15:08:52"); err != nil {
		t.Fatalf("fail execute 2nd migration: %v", err)
	}
	count, err = db.Count(&User{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected %d users in table, actual (%d)", 2, count)
	}

	if err := db.Rollback("2017-09-15:15:08:52"); err != nil {
		t.Fatalf("fail rollback 1st migration: %v", err)
	}
	var users []User
	if err := db.Select(&users); err == nil {
		t.Fatalf("user table should have been deleted")
	} else {
		var tableNotExists bool
		if mysqlError, ok := err.(*mysql.MySQLError); ok {
			if mysqlError.Number == 1146 { // table not exists error code for mysql
				tableNotExists = true
			}
		}
		if !tableNotExists {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	// check only 2 migrations left after rollback and execute
	count, err = db.Count(&SchemaMigration{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 1 {
		t.Fatalf("%d migrations should left after rollbacks, actual migrations (%d)", 2, count)
	}

	err = db.reset()
	if err != nil {
		t.Fatalf("fail reset migraitons")
	}
	if count := db.MustCount(&SchemaMigration{}); count != 0 {
		t.Fatalf("not all migrations were reseted")
	}
}
