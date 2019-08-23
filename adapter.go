package mwear

import (
	"database/sql"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/cliqueinc/mysql-wear/sqlq"
)

// Limits for db ops.
const (
	LimitInsert = 1000
)

// Map is a short representation of map[string]interface{}, used in adapter ops.
type Map map[string]interface{}

// ExecFile executes sql file.
func (db *DB) ExecFile(fileName string) error {
	filePath := getMigrationPath() + filepath.Base(fileName)
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("cannot open sql file (%s): %v", fileName, err)
	}
	defer f.Close()

	sqlData, err := ioutil.ReadAll(f)
	if err != nil {
		return fmt.Errorf("cannot read sql file (%s): %v", fileName, err)
	}

	if _, err := db.DB.Exec(string(sqlData)); err != nil {
		return err
	}
	return nil
}

// MustCreateTable ensures table is created from struct.
func (db *DB) MustCreateTable(structPtr interface{}) {
	err := db.CreateTable(structPtr)
	if err != nil {
		panic(err)
	}
}

// CreateTable creates table from struct.
func (db *DB) CreateTable(structPtr interface{}) error {
	createTableSQL := GenerateSchema(structPtr)
	if debugEnabled {
		fmt.Println(createTableSQL)
	}
	// TODO decide what to do with cmdTag aka rows created (first param)
	_, err := db.DB.Exec(createTableSQL)
	return err
}

// Adapter handles basic operations with mysql.
type Adapter struct {
	con Connection
}

type Connection interface {
	Exec(sql string, arguments ...interface{}) (res sql.Result, err error)
	Query(sql string, args ...interface{}) (*sql.Rows, error)
	QueryRow(sql string, args ...interface{}) *sql.Row
}

// Wrap wraps connection for dealing with select, insert, delete operaion
// Connection can be one of *sql.DB, *sql.Tx.
func Wrap(con Connection) *Adapter {
	return &Adapter{con}
}

// MustInsert ensures structs are inserted without errors, panics othervise.
// Limit of items to insert at once is 1000 items.
func (a *Adapter) MustInsert(structPtrs ...interface{}) sql.Result {
	res, err := a.Insert(structPtrs...)
	if err != nil {
		panic(err)
	}
	return res
}

// Insert inserts one or more struct into db. If no options specied, struct will be updated by primary key.
// Limit of items to insert at once is 1000 items.
func (a *Adapter) Insert(structPtrs ...interface{}) (sql.Result, error) {
	if len(structPtrs) == 0 {
		return nil, errors.New("nothing to insert")
	}
	if len(structPtrs) > LimitInsert {
		return nil, fmt.Errorf("insertion of more than (%d) items not allowed", LimitInsert)
	}

	var (
		model *model
		args  []interface{}
	)
	items := make([]interface{}, 0, len(structPtrs))
	for i, structPtr := range structPtrs {
		mod := parseModel(structPtr, true)
		rowModel := reflect.ValueOf(structPtr)
		items = append(items, mod)
		if i == 0 {
			model = mod
			args = make([]interface{}, 0, len(mod.Fields)*len(structPtrs))
		}

		if i != 0 && mod.TableName != model.TableName {
			return nil, errors.New("cannot insert items from different tables")
		}
		args = append(args, mod.getVals(rowModel, mod.Fields)...)
	}
	tmplData := map[string]interface{}{
		"model": model,
		"Items": items,
	}
	insertSQL := renderTemplate(tmplData, insertTemplate)
	if debugEnabled {
		fmt.Println(insertSQL)
	}

	res, err := a.con.Exec(insertSQL, args...)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// MustUpdate ensures struct will be updated without errors, panics othervise.
func (a *Adapter) MustUpdate(structPtr interface{}) {
	err := a.Update(structPtr)
	if err != nil {
		panic(err)
	}
}

// Update updates struct by primary key.
func (a *Adapter) Update(structPtr interface{}) error {
	mod := parseModel(structPtr, true)
	fieldsNoPK := mod.GetFieldsNoPK(nil)

	rowModel := reflect.ValueOf(structPtr)
	args := append(mod.getVals(rowModel, fieldsNoPK), mod.getPK(rowModel))
	updateSQL := renderTemplate(Map{"mod": mod, "fields": fieldsNoPK}, fmt.Sprintf("%s WHERE `{{.mod.PKName}}` = ?;", updateTemplate))
	if debugEnabled {
		fmt.Println(updateSQL)
	}
	_, err := a.con.Exec(updateSQL, args...)
	if err != nil {
		return err
	}

	return nil
}

// MustUpdateRows ensures rows are updated without errors, panics othervise. Returns number of affected rows.
// In case when you really need to update all rows (e.g. migration script), you need to pass mw.QueryAll() option.
// It is done to avoid unintentional update of all rows.
func (a *Adapter) MustUpdateRows(structPtr interface{}, dataMap Map, opts ...sqlq.Option) int64 {
	rowsNum, err := a.UpdateRows(structPtr, dataMap, opts...)
	if err != nil {
		panic(err)
	}

	return rowsNum
}

// UpdateRows updates rows with specified map data by query, returns number of affected rows.
// In case when you really need to update all rows (e.g. migration script), you need to pass mw.QueryAll() option.
// It is done to avoid unintentional update of all rows.
func (a *Adapter) UpdateRows(structPtr interface{}, dataMap Map, opts ...sqlq.Option) (int64, error) {
	if len(dataMap) == 0 {
		return 0, errors.New("columns for update cannot be empty")
	}
	if len(opts) == 0 {
		return 0, errors.New("query options cannot be empty")
	}

	columns := make([]string, 0, len(dataMap))
	for col := range dataMap {
		columns = append(columns, col)
	}

	mod := parseModel(structPtr, true)
	fieldsNoPK := mod.GetFieldsNoPK(columns)
	args := make([]interface{}, 0, len(dataMap))
	for _, f := range fieldsNoPK {
		val, ok := dataMap[f.MWName]
		if !ok {
			continue
		}

		args = append(args, val)
	}

	stmt, err := sqlq.Build(opts, sqlq.OpUpdate, args...)
	if err != nil {
		return 0, err
	}
	if !stmt.IsQueryAll && !strings.Contains(stmt.Query, "WHERE") {
		return 0, errors.New("query options cannot be empty")
	}

	updateTpl := updateTemplate + " " + stmt.Query + ";"
	updateSQL := renderTemplate(Map{"mod": mod, "fields": fieldsNoPK}, updateTpl)
	if debugEnabled {
		fmt.Println(updateSQL)
	}

	res, err := a.con.Exec(updateSQL, stmt.Args...)
	if err != nil {
		return 0, fmt.Errorf("update error: %v", err)
	}
	num, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("fail get number of affected rows: %v", err)
	}

	return num, nil
}

// MustSelect ensures select will not produce any error, panics othervise.
func (a *Adapter) MustSelect(structPtr interface{}, opts ...sqlq.Option) {
	err := a.Select(structPtr, opts...)
	if err != nil {
		panic(err)
	}
}

// Select performs select using query options. If no options specified, all rows will be returned.
// destSlicePtr parameter expects pointer to a slice
func (a *Adapter) Select(destSlicePtr interface{}, opts ...sqlq.Option) error {
	stmt, err := sqlq.Build(opts, sqlq.OpSelect)
	if err != nil {
		return err
	}

	mod, sliceValElement, sliceTypeElement, err := parseDestSlice(destSlicePtr)
	if err != nil {
		return err
	}
	fields := mod.getFields(stmt.Columns)
	joinMods, joinFields, err := processJoins(mod, stmt.Joins)
	if err != nil {
		return err
	}

	finalSQL := renderTemplate(Map{"mod": mod, "fields": fields, "joins": stmt.Joins, "joinFields": joinFields, "joinMods": joinMods}, selectBaseTemplate) + " " + stmt.Query + ";"
	if debugEnabled {
		fmt.Println(finalSQL)
	}

	return a.rawSelect(finalSQL, stmt.Columns, joinMods, joinFields, true, sliceValElement, sliceTypeElement, stmt.Args...)
}

func parseDestSlice(destSlicePtr interface{}) (*model, reflect.Value, reflect.Type, error) {
	var (
		defaultVal  reflect.Value
		defaultType reflect.Type
	)

	rt := reflect.TypeOf(destSlicePtr)
	if rt.Kind() != reflect.Ptr {
		return nil, defaultVal, defaultType, errors.New("please pass a pointer to slice of structs for SelectAllWhere.destSlicePtr")
	}
	rv := reflect.ValueOf(destSlicePtr)

	// This is slice itself (not a pointer to) but is in essence still a pointer
	// to elements (hence you will call sliceElement.Elem())
	sliceValElement := rv.Elem()
	sliceTypeElement := rt.Elem().Elem()

	if rt.Kind() != reflect.Ptr || sliceValElement.Kind() != reflect.Slice ||
		sliceTypeElement.Kind() != reflect.Struct {
		return nil, defaultVal, defaultType, errors.New("please pass a pointer to slice of structs for SelectAllWhere.destSlicePtr")
	}

	// Create a new instance of the slice type for model parsing to render
	// the template to create the sql!
	newThang := reflect.New(sliceTypeElement)
	mod := parseModel(newThang.Interface(), false)

	return mod, sliceValElement, sliceTypeElement, nil
}

// MustGet returns whether or not it found the item and panic on errors.
func (a *Adapter) MustGet(structPtr interface{}, opts ...sqlq.Option) bool {
	found, err := a.Get(structPtr, opts...)
	if err != nil {
		panic(fmt.Sprintf("Failed to Get item (%v)", err))
	}
	return found
}

// Get gets struct by primary key or by specified options.
func (a *Adapter) Get(structPtr interface{}, opts ...sqlq.Option) (found bool, err error) {
	getTpl := selectBaseTemplate
	var (
		query   = "WHERE `{{.mod.PKName}}` = ?"
		args    []interface{}
		columns []string
		stmt    sqlq.Query
	)
	if len(opts) != 0 {
		s, err := sqlq.Build(opts, sqlq.OpSelect)
		if err != nil {
			return false, err
		}
		stmt = *s
		query = stmt.Query
		args = stmt.Args
		columns = stmt.Columns
	}
	mod := parseModel(structPtr, true)
	fields := mod.getFields(columns)
	rowModel := reflect.ValueOf(structPtr)
	if len(opts) == 0 {
		args = []interface{}{mod.getPK(rowModel)}
	}
	if stmt.Joins != nil {
		joinMods, joinFields, err := processJoins(mod, stmt.Joins)
		if err != nil {
			return false, err
		}
		finalSQL := renderTemplate(Map{"mod": mod, "fields": fields, "joins": stmt.Joins, "joinFields": joinFields, "joinMods": joinMods}, selectBaseTemplate) + " " + stmt.Query + ";"
		if debugEnabled {
			fmt.Println(finalSQL)
		}
		sliceValElement := reflect.New(reflect.SliceOf(mod.ReflectType.Elem()))
		if err := a.rawSelect(finalSQL, stmt.Columns, joinMods, joinFields, true, sliceValElement.Elem(), mod.ReflectType.Elem(), stmt.Args...); err != nil {
			return false, err
		}
		if sliceValElement.Elem().Len() == 0 {
			return false, nil
		}

		rowModel.Elem().Set(sliceValElement.Elem().Index(0))
		return true, nil
	}

	getTpl += " " + query + ";"
	getSQL := renderTemplate(Map{"mod": mod, "fields": fields}, getTpl)
	if debugEnabled {
		fmt.Println(getSQL, args)
	}

	row := a.con.QueryRow(getSQL, args...)

	valAddrs := make([]interface{}, 0, len(fields))
	for i := range fields {
		val := rowModel.Elem().Field(fields[i].FieldPos).Addr().Interface()
		if fields[i].MWType == mw_json {
			val = &jsonScanner{val}
		} else if fields[i].Nullable {
			val = &nullScanner{rowModel.Elem().Field(fields[i].FieldPos), fields[i]}
		}
		valAddrs = append(valAddrs, val)
	}

	err = row.Scan(valAddrs...)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("scan error: %v", err)
	}

	return true, nil
}

// MustDelete ensures struct will be deleted without errors, panics othervise.
func (a *Adapter) MustDelete(structPtr interface{}) {
	if err := a.Delete(structPtr); err != nil {
		panic(err)
	}
}

// Delete deletes struct by primary key or by specified options.
func (a *Adapter) Delete(structPtr interface{}) error {
	mod := parseModel(structPtr, true)
	rowModel := reflect.ValueOf(structPtr)
	pkVal := mod.getPK(rowModel)
	if pkVal == "" {
		return fmt.Errorf("mw cant delete from table (%s), ID/PK not set", mod.TableName)
	}
	deleteSQL := renderTemplate(mod, deleteTemplate+" WHERE `{{.PKName}}` = ?")
	if debugEnabled {
		fmt.Println(deleteSQL)
	}

	_, err := a.con.Exec(deleteSQL, pkVal)
	if err != nil {
		return fmt.Errorf("delete error: %v", err)
	}

	return nil
}

// MustDeleteRows ensures rows are deleted without errors, panics othervise. Returns number of affected rows.
// In case when you really need to update all rows (e.g. migration script), you need to pass mw.QueryAll() option.
// It is done to avoid unintentional update of all rows.
func (a *Adapter) MustDeleteRows(structPtr interface{}, opts ...sqlq.Option) int64 {
	num, err := a.DeleteRows(structPtr, opts...)
	if err != nil {
		panic(err)
	}

	return num
}

// DeleteRows deletes rows by specified options. Returns number of affected rows.
// In case when you really need to update all rows (e.g. migration script), you need to pass mw.QueryAll() option.
// It is done to avoid unintentional update of all rows.
func (a *Adapter) DeleteRows(structPtr interface{}, opts ...sqlq.Option) (int64, error) {
	mod := parseModel(structPtr, true)
	stmt, err := sqlq.Build(opts, sqlq.OpDelete)
	if err != nil {
		return 0, err
	}
	if !stmt.IsQueryAll && !strings.Contains(stmt.Query, "WHERE") {
		return 0, errors.New("query options cannot be empty")
	}

	deleteTpl := deleteTemplate + " " + stmt.Query + ";"
	deleteSQL := renderTemplate(mod, deleteTpl)
	if debugEnabled {
		fmt.Println(deleteSQL)
	}

	res, err := a.con.Exec(deleteSQL, stmt.Args...)
	if err != nil {
		return 0, fmt.Errorf("delete error: %v", err)
	}
	num, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("fail get number of affected rows: %v", err)
	}

	return num, nil
}

// MustCount gets rows count, panics in case of an error.
func (a *Adapter) MustCount(model interface{}, opts ...sqlq.Option) int {
	count, err := a.Count(model, opts...)
	if err != nil {
		panic(err)
	}

	return count
}

// Count gets rows count by query.
func (a *Adapter) Count(model interface{}, opts ...sqlq.Option) (int, error) {
	type rowsCount struct {
		Count int `sql_name:"COUNT(*) as count"`
	}

	// use table of a given model
	originModel := parseModel(model, true)

	var rows []rowsCount
	mod, sliceValElement, sliceTypeElement, err := parseDestSlice(&rows)
	if err != nil {
		return 0, err
	}
	stmt, err := sqlq.Build(opts, sqlq.OpSelect)
	if err != nil {
		return 0, err
	}
	mod.TableName = originModel.TableName
	customFields := make([]*field, 0, len(mod.Fields))
	for _, f := range mod.getFields(stmt.Columns) {
		field := *f
		field.TableName = mod.TableName
		customFields = append(customFields, &field)
	}

	finalSQL := renderTemplate(Map{"mod": mod, "fields": customFields}, selectBaseTemplate)

	finalSQL += " " + stmt.Query + ";"
	if debugEnabled {
		fmt.Println(finalSQL)
	}

	if err := a.rawSelect(finalSQL, stmt.Columns, nil, nil, false, sliceValElement, sliceTypeElement, stmt.Args...); err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		return 0, nil
	}

	return rows[0].Count, nil
}

func processJoins(mod *model, joinConfigs []sqlq.JoinConfig) ([]*model, [][]*field, error) {
	if len(joinConfigs) == 0 {
		return nil, nil, nil
	}

	joins := make([]*model, 0, len(joinConfigs))
	joinFields := make([][]*field, 0, len(joinConfigs))
	for i := range joinConfigs {
		joinMod := parseModel(joinConfigs[i].StructPtr, true)
		if _, ok := mod.Joins[joinMod.ReflectType.Elem().Name()]; !ok && !joinMod.NoFields {
			return nil, nil, fmt.Errorf("unknown join relation %s, fields to be joined should be marked with tag mw:\"join\"", joinMod.ReflectType.String())
		}
		joinConfigs[i].TableName = joinMod.TableName
		if joinMod.NoFields {
			continue
		}
		joins = append(joins, joinMod)
		joinFields = append(joinFields, joinMod.getFields(joinConfigs[i].Columns))
	}

	return joins, joinFields, nil
}
