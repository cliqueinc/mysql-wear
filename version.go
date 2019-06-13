package mwear

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/cliqueinc/mysql-wear/sqlq"
)

const (
	// Set *something* as our schema version table's primary key
	schemaID       = "mw_default"
	actionInit     = "init"
	actionUpdate   = "update"
	actionStatus   = "status"
	actionRollback = "rollback"
	actionReset    = "reset"

	VersionTimeFormat = "2006-01-02:15:04:05"

	// DefaultVersion is used for keeping the default schema that we don't want to execute (but still can if needed)
	DefaultVersion = "0000-00-00:00:00:00"
)

var (
	// mh keeps all migrration data and handles migration operations.
	mh *MigrationHandler
)

// RegisterMigrationPath registers migration path for performing migration operations.
func RegisterMigrationPath(migrationPath string) error {
	if migrationPath == "" {
		return fmt.Errorf("invalid migration path: %s", migrationPath)
	}
	mh = &MigrationHandler{
		MigrationPath: migrationPath,
	}

	files, err := ioutil.ReadDir(migrationPath)
	if err != nil {
		return err
	}
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		if !strings.HasSuffix(f.Name(), ".sql") || strings.HasSuffix(f.Name(), "_down.sql") {
			continue
		}

		versionName := strings.TrimRight(f.Name(), ".sql")
		if _, err := time.Parse(VersionTimeFormat, versionName); err != nil && versionName != DefaultVersion {
			return fmt.Errorf("unrecognized version (%s) format: %v", versionName, err)
		}
		var upSQL, downSQL string

		data, err := ioutil.ReadFile(migrationPath + "/" + f.Name())
		if err != nil {
			return fmt.Errorf("fail read migration %s file: %v", versionName, err)
		}
		upSQL = string(data)
		downFileName := migrationPath + "/" + versionName + "_down.sql"
		if _, err := os.Stat(downFileName); err != nil {
			if !os.IsNotExist(err) {
				return fmt.Errorf("cannot open version (%s) down sql: %v", versionName, err)
			}
		} else {
			data, err := ioutil.ReadFile(downFileName)
			if err != nil {
				return fmt.Errorf("fail read migration %s down file: %v", versionName, err)
			}
			downSQL = string(data)
		}
		mh.RegisterMigration(versionName, upSQL, downSQL)
	}

	return nil
}

// migrationHandler handles all db migrations.
type MigrationHandler struct {
	migrations map[string]migration

	// migration paht is used to determine absolute path to each migration so mw cmd tool may be called from everywhere.
	// If not set, relative path is used to find migration file.
	MigrationPath string
}

type migration struct {
	upSQL   string
	downSQL string

	// isDefault indicates whether this migration is the default schema.
	isDefault bool
}

// RegisterMigration registers migration to process during migration update.
func (h *MigrationHandler) RegisterMigration(name string, upSQL, downSQL string) {
	if h.migrations == nil {
		h.migrations = make(map[string]migration)
	}
	if _, ok := h.migrations[name]; ok {
		panic(fmt.Sprintf("migration (%s) has already been registered", name))
	}
	h.migrations[name] = migration{upSQL: upSQL, downSQL: downSQL, isDefault: name == DefaultVersion}
}

// MigrationVersions returns slice of sorted migration versions.
func (h *MigrationHandler) MigrationVersions() []string {
	versions := make([]string, 0, len(h.migrations))
	for v := range h.migrations {
		versions = append(versions, v)
	}
	sort.Strings(versions)

	return versions
}

// SchemaMigration tracks all performed migrations.
type SchemaMigration struct {
	Version string `mw:"pk"`
	Created time.Time
}

func (SchemaMigration) TableName() string { return "mw_schema_migration" }

// MigrationLog tracks all migration activity.
type MigrationLog struct {
	Action  string
	Created time.Time
	ID      int `mw:"pk"`
	Message string
	Success bool
	Version string
}

func (mwsv MigrationLog) TableName() string { return "mw_migration_log" }

// If this wasn't successful, manually set that fact after calling this
func NewLog(action, message string, version string) *MigrationLog {
	return &MigrationLog{
		Created: time.Now().UTC(),
		Action:  action,
		Message: message,
		Version: version,
		Success: true,
	}
}

/*
Set up mw schema migration on a new pg database without mw_schema_migration
OR mw_migration_log tables. We'll fail if either exists.

Passing execDefault=true will run the default schema 000-00-00:00:00:00.sql
*/
func (db *DB) InitSchema(execDefault bool) error {
	if mh == nil {
		return errors.New("call mw.RegisterMigrationPath befoere performing any migration operation")
	}

	created, err := db.createTableIfNotExists(&SchemaMigration{})
	if err != nil {
		return fmt.Errorf("unable to init schema versioning table: %v", err)
	}
	created, err = db.createTableIfNotExists(&MigrationLog{})
	if err != nil {
		return fmt.Errorf("unable to init schema versioning log table: %v", err)
	}

	var existing []SchemaMigration
	db.MustSelect(&existing, sqlq.Order("version", sqlq.ASC))

	for _, migration := range existing {
		if _, ok := mh.migrations[migration.Version]; !ok {
			// fmt.Printf("Warning: couldn't find schema version (%s)\n", migration.Version)
		}
	}

	// Table created, insert initial log row
	schemaVersionLog := NewLog(actionInit, "Creating mw schema version and log tables", "")
	db.MustInsert(schemaVersionLog)
	// At this point the schema versioning tables should be ready to go.
	// Typically you would call UpdateSchema after this
	if created { // Can this ever be false at this point?
		// fmt.Println("Schema versioning is now initialized. Run `$ mwcmd status` for info")
		if execDefault { // If you have a default schema, you might want to run it when no migration
			// table exist yet like in the case with a new local install or new server environment
			return db.ExecuteMigration("default") // Return any associated error
		}
	}
	return nil
}

func (db *DB) createTableIfNotExists(model interface{}) (bool, error) {
	if err := db.CreateTable(model); err != nil {
		if IsTableExistsError(err) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

// UpdateSchema updates schema in case if new migrations appear.
// if execDefault param is set to true, default schema will be executed if exists.
func (db *DB) UpdateSchema(execDefault bool) error {
	if mh == nil {
		return errors.New("call mw.RegisterMigrationPath befoere performing any migration operation")
	}

	var existing []SchemaMigration
	db.MustSelect(&existing, sqlq.Order("version", sqlq.ASC))

	existingMap := make(map[string]struct{})
	for _, item := range existing {
		existingMap[item.Version] = struct{}{}
	}

	installedMigrations := make([]string, 0, len(mh.migrations))
	for _, version := range mh.MigrationVersions() {
		m, ok := mh.migrations[version]
		if !ok {
			continue
		}
		if _, ok := existingMap[version]; ok {
			continue
		}

		if m.upSQL == "" {
			return fmt.Errorf("migration (%s): up function not defined", version)
		}

		if m.isDefault && !execDefault {
			continue
		}

		migrationFunc := func(con Connection) error {
			_, err := con.Exec(m.upSQL)
			if err != nil {
				return err
			}
			_, err = Wrap(con).Insert(&SchemaMigration{Version: version, Created: time.Now().UTC()})
			if err != nil {
				return err
			}

			return nil
		}

		if err := db.execInTx(version, migrationFunc); err != nil {
			return fmt.Errorf("fail update to version (%s): %v", version, err)
		}

		installedMigrations = append(installedMigrations, version)
	}

	if len(installedMigrations) == 0 {
		// fmt.Printf("Schema is up to date\n")
		return nil
	}

	// fmt.Printf("*** Migration(s) (%s) have been installed ***\n", strings.Join(installedMigrations, ", "))
	return nil
}

// execInTx executes migration under transaction, panics on transaction errors.
func (db *DB) execInTx(version string, migrationFunc func(con Connection) error) error {
	tx, err := db.DB.Begin()
	if err != nil {
		return fmt.Errorf("fail start transaction: %v", err)
	}

	if err := migrationFunc(tx); err != nil {
		if err := tx.Rollback(); err != nil {
			panic(fmt.Sprintf("migration (%s): failed to rollback transaction: %v\n", version, err))
		}
		return err
	}
	if err := tx.Commit(); err != nil {
		panic(fmt.Sprintf("migration (%s): failed to commit transaction: %v", version, err))
	}

	return nil
}

// ExecuteMigration executes particular migration.
func (db *DB) ExecuteMigration(version string) error {
	if mh == nil {
		return errors.New("call mw.RegisterMigrationPath before performing any migration operation")
	}
	if version == "default" {
		version = DefaultVersion
	}

	m, ok := mh.migrations[version]
	if !ok {
		return fmt.Errorf("migration (%s) not found", version)
	}

	if m.upSQL == "" {
		return fmt.Errorf("migration (%s): up sql not defined", version)
	}

	migrationFunc := func(con Connection) error {
		_, err := con.Exec(m.upSQL)
		if err != nil {
			return err
		}
		schemaMigration := &SchemaMigration{}

		a := Wrap(con)
		found, err := a.Get(schemaMigration, sqlq.Equal("version", version))
		if err != nil {
			return fmt.Errorf("fail get schema migration (%s): %v", version, err)
		}
		if !found {
			_, err := a.Insert(&SchemaMigration{Version: version, Created: time.Now().UTC()})
			if err != nil {
				return err
			}
		}

		return nil
	}

	if err := db.execInTx(version, migrationFunc); err != nil {
		return fmt.Errorf("fail execute migration (%s): %v", version, err)
	}

	return nil
}

// InitMigration generates new migration file.
func InitMigration(isDefault bool) error {
	newVersion := time.Now().UTC().Format(VersionTimeFormat)
	if isDefault {
		newVersion = DefaultVersion
	}

	f, err := os.Create(getMigrationPath() + newVersion + ".sql")
	if err != nil {
		return fmt.Errorf("fail create migration sql file: %v", err)
	}
	sqlTmpl := `-- paste here migration sql code`
	if _, err := f.WriteString(sqlTmpl); err != nil {
		f.Close()
		return fmt.Errorf("fail init sql file: %v", err)
	}
	f.Close()

	f, err = os.Create(getMigrationPath() + newVersion + "_down.sql")
	if err != nil {
		return fmt.Errorf("fail create down migration sql file: %v", err)
	}
	sqlTmpl = `-- paste here migration rollback sql code`
	if _, err := f.WriteString(sqlTmpl); err != nil {
		f.Close()
		os.Remove(getMigrationPath() + newVersion + ".sql")
		return fmt.Errorf("fail init sql file: %v", err)
	}
	f.Close()

	return nil
}

// RollbackLatest rollbacks latest migration.
func (db *DB) RollbackLatest() error {
	if mh == nil {
		return errors.New("call mw.RegisterMigrationPath befoere performing any migration operation")
	}
	latestVersion := &SchemaMigration{}
	found, err := db.Get(latestVersion, sqlq.Order("version", sqlq.DESC))
	if err != nil {
		return fmt.Errorf("fail get latest schema version: %v", err)
	}
	if !found {
		// fmt.Printf("Nothing to rollback\n")
		return nil
	}
	m, found := mh.migrations[latestVersion.Version]
	if !found {
		return fmt.Errorf("migration (%s) not found", latestVersion.Version)
	}
	if err := db.rollback(latestVersion.Version, m); err != nil {
		return err
	}

	previousMigration := &SchemaMigration{}
	db.MustGet(previousMigration, sqlq.Order("version", sqlq.DESC)) // if not found, Version will be empty string, which is fine
	sLog := NewLog(
		actionRollback,
		fmt.Sprintf("Rolled back from \"%s\" to \"%s\"", latestVersion.Version, previousMigration.Version),
		latestVersion.Version,
	)
	db.MustInsert(sLog)

	// fmt.Printf("Rolled back from \"%s\" to \"%s\"\n", latestVersion.Version, previousMigration.Version)

	return nil
}

// Rollback rollbacks particular migration.
func (db *DB) Rollback(version string) error {
	if mh == nil {
		return errors.New("call mw.RegisterMigrationPath befoere performing any migration operation")
	}
	if version == "default" {
		version = DefaultVersion
	}
	schemaMigration, found := mh.migrations[version]
	if !found {
		return fmt.Errorf("migration (%s) not found", version)
	}
	if err := db.rollback(version, schemaMigration); err != nil {
		return err
	}

	sLog := NewLog(
		actionRollback,
		fmt.Sprintf("Rolled back migration \"%s\"", version),
		version,
	)
	db.MustInsert(sLog)

	// fmt.Printf("Rolled back migration \"%s\"\n", version)

	return nil
}

// Reset resets all migration data.
func (db *DB) Reset() error {
	_, err := db.DeleteRows(&SchemaMigration{}, sqlq.All())
	if err != nil {
		return err
	}

	sLog := NewLog(actionReset, "Reset all data", "")
	db.MustInsert(sLog)

	// fmt.Print("Migation data has been reset\n")
	return nil
}

// rollback rollbacks particular migration.
func (db *DB) rollback(version string, m migration) error {
	if mh == nil {
		return errors.New("call mw.RegisterMigrationPath befoere performing any migration operation")
	}

	if m.downSQL == "" {
		return fmt.Errorf("migration (%s): down sql not found", version)
	}

	migrationFunc := func(con Connection) error {
		_, err := con.Exec(m.downSQL)
		if err != nil {
			return err
		}
		if _, err := Wrap(con).DeleteRows(&SchemaMigration{}, sqlq.Equal("version", version)); err != nil {
			return err
		}

		return nil
	}

	if err := db.execInTx(version, migrationFunc); err != nil {
		return fmt.Errorf("fail rollback version (%s): %v", version, err)
	}

	return nil
}

// TODO print the version and latest X logs in a less retarded way
func (db *DB) PrintVersionStatus() error {
	logs := []MigrationLog{}
	if err := db.Select(&logs, sqlq.Order("created", sqlq.DESC), sqlq.Limit(10)); err != nil {
		return fmt.Errorf("can't get latest logs: %v", err)
	}

	latestMigrations := []SchemaMigration{}
	if err := db.Select(&latestMigrations, sqlq.Order("version", sqlq.DESC), sqlq.Limit(10)); err != nil {
		return fmt.Errorf("can't get latest migrations: %v", err)
	}

	fmt.Println("--------------------------------------------------")
	fmt.Println("Last 10 schema change logs")
	fmt.Println("--------------------------------------------------")
	// common.PrettyPrintJson(logs)

	fmt.Println("--------------------------------------------------")
	fmt.Println("Latest migrations info")
	fmt.Println("--------------------------------------------------")
	if len(latestMigrations) != 0 {
		for i := range latestMigrations {
			fmt.Println(latestMigrations[len(latestMigrations)-i-1].Version)
		}
		fmt.Println("--------------------------------------------------")
	} else {
		fmt.Printf("No migrations so far\n")
		fmt.Println("--------------------------------------------------")
	}

	return nil
}

func getMigrationPath() string {
	if mh != nil && mh.MigrationPath != "" {
		return filepath.Clean(mh.MigrationPath) + "/"
	}

	return "./"
}
