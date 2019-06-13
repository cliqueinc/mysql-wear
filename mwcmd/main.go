package main

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	mw "github.com/cliqueinc/mysql-wear"
)

const (
	DB_USE_TLS        = "DB_USE_TLS"
	DB_NAME           = "DB_NAME"
	DB_USER           = "DB_USER"
	DB_PASSWORD       = "DB_PASSWORD"
	DB_PORT           = "DB_PORT"
	DB_HOST           = "DB_HOST"
	DB_MIGRATION_PATH = "DB_MIGRATION_PATH"
)

var db *mw.DB

/*
How will be used:
- Install this binary
- This bin is looking in the project home for a configurable migrations folder like
 schema_updates/1,2,3,4,5
*/

func help() {
	fmt.Println(`Basic Commands: "$ mwcmd up|init|rollback|status"`)
	fmt.Println(`Generate init: "$ mwcmd gen init StructName shortName"`)
	fmt.Println("")
	fmt.Println("For commands other than gen, pass -d for debug query logging")
	os.Exit(-1)
}

// RunCmd runs mw commands with args.
// mw is usually bind to some environment and mw migrations
func main() {
	args := os.Args
	if len(args) < 2 {
		help()
	}

	action := args[1]
	arg2 := ""

	if len(args) > 2 {
		arg2 = args[2]
		if arg2 == "-d" {
			// GetConfig().LogQueries = true
			if len(args) > 3 && !strings.HasPrefix(args[3], "-") {
				arg2 = args[3]
			}
		}
	}

	var err error
	switch action {
	case "up":
		execDefault := arg2 == "--exec-default"
		err = getDB().UpdateSchema(execDefault)
	case "new-migration":
		isDefault := len(args) > 2 && args[2] == "default"
		err = mw.InitMigration(isDefault)
	case "exec":
		if arg2 == "" {
			err = errors.New("please specify migration name to execute")
			break
		}
		err = getDB().ExecuteMigration(arg2)
	case "init":
		// TODO do we want to allow an additional flag for initing with default?
		err = getDB().InitSchema(false)
	case "reset":
		err = getDB().Reset()
	case "gen":
		err = Generate(args)
	case "rollback":
		if arg2 != "" && !strings.HasPrefix(arg2, "-") {
			err = getDB().Rollback(arg2)
		} else {
			err = getDB().RollbackLatest()
		}
	case "status":
		err = getDB().PrintVersionStatus()
	case "help":
		if db != nil {
			db.DB.Close()
		}
		help()
	default:
		if db != nil {
			db.DB.Close()
		}
		help()
	}

	if db != nil {
		db.DB.Close()
	}
	if err != nil {
		fmt.Printf("Failed: %v\n", err)
		os.Exit(-1)
	}
	os.Exit(0)
}

func getDB() *mw.DB {
	if db != nil {
		return db
	}
	cfg, err := getMySQLConfigFromEnv()
	if err != nil {
		fmt.Printf("cannot get mysql configs from env: %v\n", err)
		fmt.Println(`db connection is initialized from following env vars:`)
		fmt.Println("DB_NAME,DB_USER,DB_PASSWORD,DB_PORT,DB_HOST,DB_USE_TLS,DB_MIGRATION_PATH")
		os.Exit(-1)
	}

	dbCon, err := openDB(cfg)
	if err != nil {
		fmt.Printf("fail init DB: %v\n", err)
		os.Exit(-1)
	}

	if err := mw.RegisterMigrationPath(cfg.DBMigrationPath); err != nil {
		fmt.Printf("fail register migration path: %v\n", err)
		dbCon.DB.Close()
		os.Exit(-1)
	}

	db = dbCon
	return dbCon
}

/*
See readme for details on generation
*/
func Generate(args []string) error {
	// Right now we only support `mwcmd gen init StructName`
	if len(args) != 5 || args[2] != "init" {
		return errors.New("gen command currently only supports `$ mwcmd gen init StructName shortName`")
	}
	fmt.Println(mw.GenerateInit(args[3], args[4]))
	return nil
}

type config struct {
	Host            string
	Port            int
	User, Password  string
	UseTLS          bool
	DBName          string
	DBMigrationPath string
}

func getMySQLConfigFromEnv() (config, error) {
	var useTLS bool
	if strings.ToLower(os.Getenv(DB_USE_TLS)) == "true" {
		useTLS = true
	}
	port, _ := strconv.Atoi(os.Getenv(DB_PORT))
	if port <= 0 {
		port = 3306
	}
	host := "127.0.0.1"
	if hostEnv := os.Getenv(DB_HOST); host != "" {
		host = hostEnv
	}

	cfg := config{
		DBName:          os.Getenv(DB_NAME),
		User:            os.Getenv(DB_USER),
		Password:        os.Getenv(DB_PASSWORD),
		Host:            host,
		Port:            port,
		UseTLS:          useTLS,
		DBMigrationPath: os.Getenv(DB_MIGRATION_PATH),
	}
	if cfg.DBName == "" {
		return config{}, errors.New("DB_NAME required")
	}
	if cfg.DBMigrationPath == "" {
		return config{}, errors.New("DB_MIGRATION_PATH required")
	}
	if _, err := os.Stat(cfg.DBMigrationPath); os.IsNotExist(err) {
		return config{}, fmt.Errorf("directory (%s) not exists", cfg.DBMigrationPath)
	}
	if cfg.User == "" {
		cfg.User = "root"
	}

	return cfg, nil
}

// openDB inits db connection from config.
func openDB(cfg config) (*mw.DB, error) {
	tls := "false"
	if cfg.UseTLS {
		tls = "true"
	}

	conPath := fmt.Sprintf("tcp(%s:%d)", cfg.Host, cfg.Port)
	dsn := fmt.Sprintf(
		"%s:%s@%s/%s?tls=%s&parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci&sql_mode=''",
		cfg.User,
		cfg.Password,
		conPath,
		cfg.DBName,
		tls,
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("fail open db connection: %v", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("fail ping db: %v", err)
	}

	return mw.New(db), nil
}
