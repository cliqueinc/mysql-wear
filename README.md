# MySQL Wear Specification

## Quickstart

Before you start using mysql-wear package, you need to init mysql connection,
like in this test sample:

```golang
import (
  mw "github.com/cliqueinc/mysql-wear"
  "database/sql"
)

func main() {
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
```

<b>Important!</b> Make sure db connection has an option `parseTime=true`, othervise time.Time struct will not be parsed correctly!

Then, we can initialize mysql-wear wrapper for dealing with CRUD mysql operations:

```golang
db := mw.New(mysqlDB)
```

Also, instead of using whole \*sql.DB instance, we can wrap single sql connection for using
mysql-wear helpers:

```golang
tx, err := db.Begin()
if err != nil {
  t.Fatalf("cannot begin transaction: %s", err)
}

// now its possible to use crud operations against transaction connection!
a := Wrap(tx)
res, err := a.Insert(fu)
if err != nil {
  tx.Rollback()
  log.Fatalf("failed to insert row: %s", err)
}
num, err := res.RowsAffected()
if err != nil {
  tx.Rollback()
  log.Fatalf("%v", err)
}
if err := tx.Commit(); err != nil {
  log.Fatal(err)
}
```

## Using Code Generation To Get Started

You can use the following example script to generate a simple service
for use with `mw`. You should put this code into a main.go and run it
using `go run main.go`:

```bash
mwcmd gen init Article a
```

This command tells mwcmd to generate file that produces sql migration, crud, and crud test files for a specified struct. `Article` - name of a struct, `a` - short name that will be used in code.
Command produces following code:

```golang
package main

import (
  "fmt"
  mw "github.com/cliqueinc/mysql-wear"
)

type Article struct {
  ID          string // IMPORTANT: this field required or field with tag `mw:"pk"`.
  Name        string
  Description string
}

func main() {
  fmt.Println(mw.GenerateSchema(&Article{}))
  fmt.Println(mw.GenerateModel(&Article{}, "a"))
  fmt.Println(mw.GenerateModelTest(&Article{}, "a"))
}
```

- `mw.GenerateSchema` - prints sql code for creating a table for the specified model.
- `mw.GenerateModel` - prints golang model crud file.
- `mw.GenerateModelTest` - prints golang test for current model.

Normally when you are ready to create a new data model, you can first
design and create your struct, then pop it into an example file like this
then build out the service or supporting code based on the generated code.

- Note you will need to copy the struct into the example service file.

## Naming

All sql queries usually have 2 methods: first starts with `Must` (MustInsert) and the second just the naming of a method (Insert), where `MustInsert` panics in case of an error, and `Insert` returns an error. First advantage is readability, it's a common naming style in go (like `MustExec` in template lib), and newcomer usually awares that `MustInsert` may panic, while `Insert` returns just an error.

```golang
found, err := db.Get(&user)
if err != nil {
  panic(err)
}
if !found { // user not exist
  db.MustInsert(&user)
}
```

Sometimes helpers may be used to fetch additional info in case of an error:

```golang
err = db.Insert(f2)
if mw.IsUniqueViolationError(err) {
  return errors.New("user with the same email already exists") // supposing email is unique for each row
}
```

## Struct Tags

- `mw`

  - `mw:"pk"` detects whether a field is a primary key. If not such tag set, mw will set `ID` field as primary.
  - `mw:"nullable"` tells mw that a field can be `NULL`.
  - `mw:"-"` tells mw to skip this field from all sql operations.

  <strong>Gotchas:</strong>

  - It is strongly encouraged for your models to have ONLY one of the following (to define the primary key):
    - A `ID string` field OR
    - A `` UserID string `mw:"pk"` `` field where UserID can be whatever
    - Having both or none will cause an error
  - Custom time.Time fields are not supported. Your models must use time.Time directly.
  - If you have custom times for json output see [this](http://choly.ca/post/go-json-marshalling/)

- `sql_name`
  By default mw converts struct name (usually CamelCased) into underscored name. But sometimes we have such field names like
  `RedirectURL`, which may be converted in not a proper way, so all one needs to do is to add `sql_name` tag:

  ```golang
  type blog struct {
    ID string
    RedirectURL string `sql_name:"redirect_url"`
  }
  ```

## Insert

There are 2 methods: Insert(structPtrs ...interface{}) and MustInsert(structPtrs ...interface{}). Multiple items from the same struct may be inserted
per one sql query. Attempt to call insert without models or more than `1000` items, or insert models that are different structs will cause an error.

```golang
u1 := &User{
  ID:   "id1",
  Name: "John",
}
u2 := &User{
  ID:   "id2",
  Name: "Forest",
}

db.MustInsert(u1, u2)
```

In case struct's primary key is int, its possible to get inserted id from sql result:

```golang
type blog struct {
  ID int64
  Name string
}
b := &blog{ Name: "my blog" }
res := db.MustInsert(b)
id, err := res.LastInsertId()
if err != nil {
  log.Fatalf("fail get last id: %v", err)
}
b.ID = id
```

## Select

The idea is that we usually use the same patterns for building raw queries, such as limit, ordering, IN construction, where, etc. The purpose of method is to simplify quering, which can make using mw more fun.

For example, in order to query by IN with limit 5 ordering by `created` column, one'd need to type:

```golang
err := db.Select(
  &users,
  sqlq.IN("id", 111, 222, 333),
  sqlq.Limit(5),
  sqlq.Order("created", sqlq.DESC),
)
```

Or in case of more complicated conditions:

```golang
db.MustSelect(
  &blogs,
  sqlq.OR(
    sqlq.Equal("name", "blog4"),
    sqlq.AND(
      sqlq.Equal("descr", "descr3"),
      sqlq.IN("id", blog1ID, blog2ID),
    ),
  ),
)
```

The method is implemented using functional options pattern, which is super lightweight and is easy extendable for adding common constructions.

Example:

```golang
opts := []sqlq.Option{
  sqlq.Order("id", sqlq.ASC),
  sqlq.Limit(50),
  sqlq.Offset(100),
}

if excludeEmail {
  opts = append(opts, sqlq.NotEqual("email", "some@email"))
}

db.MustSelect(&users, opts...)
```

### <strong>Default limit</strong>

If no limit specified for select, the default limit will be added (`1000`). If you <strong>really need</strong> to fetch all rows, you need to
add sqlq.All() option:

```golang
db.MustSelect(&blogs, sqlq.Order("updated", sqlq.DESC), sqlq.All())
```

## Select specific columns

In case one needs to fetch only custom columns (for example table have a column html_content, which is too expensive to load each time), they can simply use `sqlq.Columns` query option:

```golang
var users []User
db.MustSelect(&users, sqlq.Columns("id", "name", "salary"), sqlq.Limit(2), sqlq.GreaterThan("salary", 500))

for _, u := range users {
  fmt.Printf("user (%s) has salary (%d) (not secret anymore!)\n", u.Name, u.Salary))
}
```

## Get

Get is almost the same as select except it returns exactly 1 row and returns flag whether row exists and an error if some has occured.

```golang
user := &user{ID: "111"}
found, err := db.Get(&user) // by default gets by primary key
if err != nil {
  return fmt.Error("db error: %v", err)
}
if !found {
  return errors.New("user not found")
}

user2 := &user{}
found := db.MustGet(&user2, sqlq.Equal("email", "user2@mail.com")) // this one will fetch by email
if !found {
  return errors.New("user not found by email")
}

user3 := &user{ID: "333"}
// if only few fields needed form db
found := db.MustGet(&user3, sqlq.Columns("company_id")) // by default gets by primary key
if !found {
  return errors.New("user not found")
}
fmt.Println(user3.CompanyID)
```

## Update

Update updates struct by primary key

```golang
if err := db.Update(&user); err != nil {
  return fmt.Errorf("fail update user: %v", err)
}
```

## UpdateRows

It is also posible to update multiple rows at once:

```golang
num, err := db.UpdateRows(&user{}, mw.Map{"scores": 50, "is_active": false}, sqlq.Equal("company_id", "555"), sqlq.NotEqual("is_active", false))
if err != nil {
  return fmt.Errorf("db error: %v", err)
}
```

- `&user{}`, first argument, is a struct that represents table for update, mw greps metadata from it.
- `mw.Map` is an alias for `map[string]interface{}`, represents updated data
- `sqlq.Equal("company_id", "555")`, `sqlq.NotEqual("is_active", false)` - optional query options for updating rows.

The sample above will produce something like:

```sql
UPDATE `user` SET `scores`=50, `is_active`=fase WHERE `company_id`="555" AND `is_active` != false;
```

### <b>Update all rows</b>

By default, if you try to call `db.UpdateRows` without any query option, it will produce an error: `query options cannot be empty`

In case when you <b>really need</b> to update all rows (e.g. migration script), you need to pass `sqlq.All()` option.
It is done to avoid unintentional update of all rows:

```golang
num, err := db.UpdateRows(&user{}, mw.Map{"is_active": false}, sqlq.All())
if err != nil {
  return err
}
fmt.Println(num)
```

## Delete

Delete deletes struct by primary key

```golang
if err := db.Delete(&user); err != nil {
  return fmt.Errorf("fail delete user: %v", err)
}
```

## DeleteRows

It is also posible to delete multiple rows at once:

```golang
num, err := db.DeleteRows(&user{}, sqlq.Equal("company_id", "555"), sqlq.NotEqual("is_active", false))
if err != nil {
  return fmt.Errorf("db error: %v", err)
}
```

- `&user{}`, first argument, is a struct that represents table, mw greps metadata from it.
- `sqlq.Equal("company_id", "555")`, `sqlq.NotEqual("is_active", false)` - optional query options for deleting rows.

The sample above will produce something like:

```sql
DELETE FROM "user" WHERE "company_id"="555";
```

### <b>Delete all rows</b>

By default, if you try to call `db.DeleteRows` without any query option, it will produce an error: `query options cannot be empty`

In case when you <b>really need</b> to delete all rows (e.g. migration script), you need to pass `sqlq.All()` option.
It is done to avoid unintentional deleting of all rows:

```golang
num, err := db.DeleteRows(&user{}, sqlq.All())
if err != nil {
  return err
}
fmt.Println(num)
```

## Count

In order to get count of all rows by query, just call something like a sample below:

```golang
count := db.MustCount(&user{}, sqlq.LessThan("score", 1000))
fmt.Printf("found %d rows\n", count)
```

## Generate sum Codez!

The easiest way to understand how to use the code generator is to view examples/generate_print.go

Then run it via `go run examples/generate_print.go`

It will print out the sql create table statement and a stub model and test.

You can use the following two step process to create everything from scratch:

Run the command `mwcmd gen init StructName shortStructName` where StructName is something like "User" and
shortStructName could be "user" and is the lowercase self/this reference for class methods
(will make more sense in a second).

`mwcmd gen init User user`

This will print a main go program. Place this code in a go file called main.go. You should then spend some
time and fill out the struct and look over this code. Once the struct looks good, run:

`go run main.go`

This will print a create table statement, sample struct methods, and a simple test stub. The create table should
go in a new up.sql in a new schema migration folder (see instructions below). From the user example, you would
put the struct and methods in a model_user.go file and the test in a model_user_test.go file.

Delete your main.go file now and start building out your model/service!

## Versioning (schema migrations)

In order to use the schema migration capabilities, you need to follow next steps:

### 1. Install mwcmd tool

```bash
go install github.com/cliqueinc/mysql-wear/mwcmd
```

So now you can operate migrations.

#### Before using

`mwcmd` tool initializes db connection from ENV vars:

- DB_NAME - name of the database, required.
- DB_MIGRATION_PATH - path to migrations directory, required. Example: `./db/migrations`
- DB_USER - db user name, root by default.
- DB_PASSWORD - db password.
- DB_PORT - db port, 3306 by default
- DB_HOST - db host, 127.0.0.1 by default
- DB_USE_TLS - whether to use tls, false by default

IMPORTANT: don't forget to run next command (to initialize migrations tables in db):

```bash
mwcmd init
```

Example usage:

```bash
DB_PASSWORD=root DB_PORT=3406 DB_MIGRATION_PATH=./migrations DB_NAME=mw_tmp mwcmd status
```

### 2. Init new migration

mwcmd inits mysql connection from default env vars, and migration path also taken from `DB_MIGRATION_PATH` variable.

In order to init migration:

```bash
mwcmd new-migration
```

This command creates 2 migration files (like `2017-09-19:15:08:52.sql` and `2017-09-19:15:08:52_down.sql`), up and rollback sql commands, the second one
is optional and can be safely deleted.

If you are running migration by your app, you need to register the migration path before mw init:

```golang
import (
	"github.com/cliqueinc/mw"
)

mw.RegisterMigrationPath(cfg.AppPath+"db/migrations")
```

So the migration handler knows where to take migration from.
Now mw nows where to take migrations from, and you are able to call mw.InitSchema(), or db.UpdateSchema():

```golang
if err := mw.InitSchema(false); err != nil {
  panic(fmt.Sprintf("Fail init schema: %v\n", err))
}
```

### <strong>mw commands</strong>

In order to deal with mysql migration, you can install `mwcmd` tool:

- go to `mwcmd` directory, then execute:

```bash
go install
```

- #### mwcmd init

  Creates required mw schema tables.
  `mwcmd init` also ensures all executed migrations exist in corresponding files.

- #### mwcmd up

  Checks is there any migration that needs to be executed, and executes them in ascending order.

- #### mwcmd new-migration [default]

  Generates new migration file, see `Migration file internals` for more info. If the next argument is `default`, the migration 0000-00-00:00:00:00.sql
  will be generated. It won't be added to migration log and won't be executed unless you explicitly call `exec default`. It is made in order to have
  an ability to keep existing schema, which needs to be executed only once, and most likely, only in local environment.

- #### mwcmd status

  Prints latest migration logs and most recent migrations.

- #### mwcmd exec [version]

  Executes specific migration and markes it as applied if needed. Type `mwcmd exec default`, if you want to execute the default migration.

- #### mwcmd rollback [version]

  Rollbacks specific version. If no version specified, just rollbacks the latest one.

- #### mwcmd gen

  Run the command mwcmd gen init StructName shortStructName where StructName is something like "User" and shortStructName could be "user" and is the lowercase self/this reference for class methods (will make more sense in a second).

  mwcmd gen init User user

  This will print a main go program. Place this code in a go file called main.go. You should then spend some time and fill out the struct and look over this code. Once the struct looks good, run:

  go run main.go

  This will print a create table statement, sample struct methods, and a simple test stub. The create table should go in a new up.sql in a new schema migration folder (see instructions below). From the user example, you would put the struct and methods in a model_user.go file and the test in a model_user_test.go file.

  Delete your main.go file now and start building out your model/service!
