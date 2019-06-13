package mwear

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/go-sql-driver/mysql"
)

const (
	mw_pk_string = "VARCHAR(255) NOT NULL PRIMARY KEY"
	mw_pk_int    = "INT NOT NULL AUTO_INCREMENT PRIMARY KEY"
	mw_date_time = "timestamp {nullable}"
	mw_json      = "JSON"
	mw_small_int = "SMALLINT {nullable} DEFAULT 0"
	mw_integer   = "INT {nullable} DEFAULT 0"
	mw_boolean   = "tinyint(1) {nullable} DEFAULT 0"
	mw_float     = "DOUBLE {nullable} DEFAULT 0"
	mw_text      = "VARCHAR(255) {nullable} DEFAULT ''"
)

func init() {
	cachedModelMap.Init()
}

type model struct {
	Struct     interface{}
	StructName string

	// Used to generate the model definition
	ShortName string
	TableName string

	Fields []*field

	ReflectType reflect.Type

	// Explicitly store the PK name for where clauses
	// Note PKName will always be id for now
	PKName string
	// PKPos is a position of a primary key.
	PKPos int

	// used if we don't want to fetch model's fields
	NoFields bool

	// Joins maps joined table name to joined field position.
	Joins map[string]int
}

func (mod *model) IsIntPK() bool {
	switch mod.GetPKKind() {
	case reflect.Int, reflect.Int32, reflect.Int64:
		return true
	default:
		return false
	}
}

func (mod *model) GetPKField() *field {
	for _, f := range mod.Fields {
		if f.MWName == mod.PKName {
			return f
		}
	}
	return nil
}

func (mod *model) GetPKKind() reflect.Kind {
	if pkField := mod.GetPKField(); pkField != nil {
		return pkField.ReflectType.Kind()
	}
	return reflect.Invalid
}

type modelSyncMap struct {
	modelMap map[string]*model
	mux      sync.Mutex
}

func (mm *modelSyncMap) Get(name string) (*model, bool) {
	val, ok := mm.modelMap[name]
	return val, ok
}

func (mm *modelSyncMap) Init() {
	mm.mux.Lock()
	mm.modelMap = make(map[string]*model)
	mm.mux.Unlock()
}

func (mm *modelSyncMap) Set(name string, m *model) {
	mm.mux.Lock()
	mm.modelMap[name] = m
	mm.mux.Unlock()
}

var cachedModelMap modelSyncMap

// In cases like UPDATE we need to get the list of fields sans ID since you can't update PK.
// Must be exported since the templates call this.
func (mod *model) GetFieldsNoPK(columns []string) []*field {
	fields := mod.getFields(columns)
	filteredFields := make([]*field, 0, len(fields))
	for _, f := range fields {
		if f.MWName == mod.PKName {
			continue
		}
		filteredFields = append(filteredFields, f)
	}

	return filteredFields
}

func (mod *model) getFields(columns []string) []*field {
	if len(columns) == 0 {
		return mod.Fields
	}
	fields := make([]*field, 0, len(columns))
	for i := range mod.Fields {
		if i == mod.PKPos {
			fields = append(fields, mod.Fields[i])
			continue
		}

		var includeColumn bool
		for _, col := range columns {
			if col == mod.Fields[i].MWName {
				includeColumn = true
				break
			}
		}

		if includeColumn {
			fields = append(fields, mod.Fields[i])
		}
	}
	if len(columns) != len(fields) { // some column not exists in db
	ColumnsLoop:
		for _, col := range columns {
			for _, f := range fields {
				if f.MWName == col {
					continue ColumnsLoop
				}
			}
			panic(fmt.Sprintf("unrecognized column (%s)", col))
		}
	}

	return fields
}

// Get a slice of the vals for interfacing with sql
func (pm *model) getVals(rowModel reflect.Value, fields []*field) []interface{} {
	vals := make([]interface{}, 0, len(fields))
	for _, f := range fields {
		fieldVal := reflect.Indirect(rowModel).Field(f.FieldPos).Interface()
		if f.MWType == mw_json {
			v, err := json.Marshal(fieldVal)
			if err != nil {
				if debugEnabled {
					fmt.Println("marshal failed!", err)
				}
			}
			fieldVal = v
		}
		vals = append(vals, fieldVal)
	}
	return vals
}

/*
GoType are the reflect types so you can do == reflect.Ptr or whatever

MWType will be quite limited initially. Basic ints, floats, only text (no need to use varchar with modern pg), lots of jsonb
Will also include the constraints for pk and all not null initially.

*/
type field struct {
	TableName    string
	GoName       string
	MWName       string
	MWType       string
	ReflectType  reflect.Type
	ReflectValue reflect.Value
	ReflectKind  reflect.Kind

	// FieldPos is a position of a field in our struct.
	FieldPos int

	mwNameQuoted       string
	mwNameQuotedSelect string
	joinedMWName       string

	// Nullable specifies whether a field can be NULL.
	Nullable bool
}

func (f *field) MWNameQuoted() string {
	if f.mwNameQuoted != "" {
		return f.mwNameQuoted
	}
	trimString := func(str string) string {
		return strings.Replace(strings.Replace(str, ";", "", -1), "`", "", -1)
	}
	escapeString := func(str string) string {
		return "`" + trimString(str) + "`"
	}
	f.mwNameQuoted = escapeString(f.MWName)

	return f.mwNameQuoted
}

func (f *field) MWNameQuotedSelect() string {
	if f.mwNameQuotedSelect != "" {
		return f.mwNameQuotedSelect
	}
	trimString := func(str string) string {
		return strings.Replace(strings.Replace(str, ";", "", -1), "`", "", -1)
	}
	escapeString := func(str string) string {
		return "`" + trimString(str) + "`"
	}

	mwName := f.MWName
	parts := strings.Split(strings.ToLower(mwName), " as ")
	if len(parts) > 1 {
		if len(parts) == 2 {
			mwName = trimString(parts[0]) + " as " + escapeString(parts[1])
		} else {
			panic(fmt.Sprintf("invalid column name (%s)", mwName))
		}
	} else {
		mwName = "`" + f.TableName + "`." + escapeString(mwName)
	}

	f.mwNameQuotedSelect = mwName
	return f.mwNameQuotedSelect
}

func (f *field) JoinedMWName() string {
	if f.joinedMWName != "" {
		return f.joinedMWName
	}
	trimString := func(str string) string {
		return strings.Replace(strings.Replace(str, ";", "", -1), "`", "", -1)
	}
	escapeString := func(str string) string {
		return "`" + trimString(str) + "`"
	}

	mwName := "`" + f.TableName + "`." + escapeString(f.MWName)
	return mwName
}

type nullScanner struct {
	fieldVal reflect.Value
	field    *field
}

var timeType = reflect.TypeOf(time.Time{}).String()

func (scanner *nullScanner) Scan(val interface{}) error {
	if val == nil {
		return nil
	}

	var fs sql.Scanner
	switch scanner.field.ReflectKind {
	case reflect.String:
		fs = &sql.NullString{}
	case reflect.Int8, reflect.Int16, reflect.Int, reflect.Int32, reflect.Uint, reflect.Uint32, reflect.Uint8,
		reflect.Int64, reflect.Uint64, reflect.Uint16:
		fs = &sql.NullInt64{}
	case reflect.Float32, reflect.Float64:
		fs = &sql.NullFloat64{}
	case reflect.Bool:
		fs = &sql.NullBool{}
	case reflect.Struct, reflect.Array, reflect.Slice:
		if scanner.field.ReflectType.String() == timeType {
			fs = &mysql.NullTime{}
		} else {
			fs = &jsonScanner{scanner.fieldVal.Addr().Interface()}
		}
	default:
		panic(fmt.Sprintf("cannot scan nullable field of type %s", scanner.field.ReflectKind))
	}

	if err := fs.Scan(val); err != nil {
		return err
	}

	switch sv := fs.(type) {
	case *sql.NullString:
		scanner.fieldVal.SetString(sv.String)
	case *sql.NullInt64:
		switch scanner.field.ReflectKind {
		case reflect.Int8, reflect.Int16, reflect.Int, reflect.Int32, reflect.Int64:
			scanner.fieldVal.SetInt(sv.Int64)
		case reflect.Uint, reflect.Uint32, reflect.Uint8, reflect.Uint64, reflect.Uint16:
			scanner.fieldVal.SetUint(uint64(sv.Int64))
		}
	case *sql.NullFloat64:
		scanner.fieldVal.SetFloat(sv.Float64)
	case *sql.NullBool:
		scanner.fieldVal.SetBool(sv.Bool)
	case *mysql.NullTime:
		scanner.fieldVal.Set(reflect.ValueOf(sv.Time))
	}

	return nil
}

func (fi *field) isPrimaryKey() bool {
	if strings.Contains(strings.ToLower(fi.MWType), "primary key") {
		return true
	}
	return false
}

func (mod *model) setTableName(rowModel reflect.Value) {
	tableNameMethod := rowModel.MethodByName("TableName")

	if tableNameMethod.IsValid() {
		vals := tableNameMethod.Call([]reflect.Value{})
		mod.TableName = vals[0].String()
	} else {
		if mod.StructName == "" { // Just in case...
			panic("Cannot call setTableName until StructName is set")
		}
		mod.TableName = parseName(mod.StructName)
	}
}

func (mod *model) getPK(rowModel reflect.Value) string {
	if mod.PKPos == -1 {
		panic(fmt.Sprintf("Missing primary key for table (%s)", mod.TableName))
	}
	pkType := mw_pk_string
	for _, f := range mod.Fields {
		if f.MWName == mod.PKName {
			pkType = f.MWType
			break
		}
	}

	if pkType == mw_pk_int {
		return strconv.FormatInt(reflect.Indirect(rowModel).Field(mod.PKPos).Int(), 10)
	}
	return reflect.Indirect(rowModel).Field(mod.PKPos).String()
}

func parseModel(mm interface{}, requirePK bool) *model {
	modType := reflect.TypeOf(mm)
	typeName := modType.String()
	if mod, ok := cachedModelMap.Get(typeName); ok {
		return mod
	}

	mod := &model{
		Struct:      mm,
		ReflectType: modType,
	}
	modKind := modType.Kind()
	rowModel := reflect.ValueOf(mm)

	if modKind != reflect.Ptr || rowModel.Elem().Kind() != reflect.Struct {
		panic("Please pass a struct pointer to parseModel")
	}
	elem := rowModel.Elem()
	elemType := elem.Type()
	mod.StructName = elemType.Name()

	mod.setTableName(rowModel)

	fieldLen := elem.NumField()
	mod.Fields = make([]*field, 0, fieldLen)
	for i := 0; i < fieldLen; i++ {
		fieldName := elemType.Field(i).Name

		// Get the mw struct tag for this field
		tagValue := strings.TrimSpace(elemType.Field(i).Tag.Get("mw"))
		if tagValue == "-" {
			continue
		}
		fieldType := elemType.Field(i).Type
		fieldKind := fieldType.Kind()

		if tagValue == "join" {
			if mod.Joins == nil {
				mod.Joins = make(map[string]int)
			}
			var joinType string
			elType := fieldType
			if fieldKind == reflect.Slice {
				elType = fieldType.Elem()
			}
			if fieldKind == reflect.Ptr {
				joinType = elType.Elem().Name()
			} else {
				joinType = elType.Name()
			}
			mod.Joins[joinType] = i
			continue
		}
		// reserved field name
		if fieldName == "MW" {
			if tagValue == "many_to_many" {
				mod.NoFields = true
			}
			continue
		}

		var mwName string
		if tagName := strings.TrimSpace(elemType.Field(i).Tag.Get("sql_name")); tagName != "" {
			mwName = tagName
		} else {
			mwName = parseName(fieldName)
		}

		// Support PK and - struct tags for now
		newField := &field{
			TableName:   mod.TableName,
			GoName:      fieldName,
			MWName:      mwName,
			ReflectKind: fieldKind,
			ReflectType: fieldType,
			FieldPos:    i,
		}
		if tagValue == "nullable" {
			newField.Nullable = true
		}

		newField.setMWType(mod, tagValue)
		if newField.MWName == mod.PKName {
			mod.PKPos = i
		}

		mod.Fields = append(mod.Fields, newField)
	}
	// TODO we really need to do more inspection of the model to make sure there isn't
	// more than one PK and/or warn about ID field in addition to PK
	if requirePK && mod.PKName == "" {
		panic(fmt.Sprintf("Missing primary key for table (%s)", mod.TableName))
	}
	cachedModelMap.Set(typeName, mod)

	return mod
}

func (fi *field) setMWType(mod *model, tagVal string) {

	switch tagVal {
	case "pk":
		mod.PKName = fi.MWName
		switch fi.ReflectType.Kind() {
		case reflect.Int, reflect.Int32, reflect.Int64:
			fi.MWType = mw_pk_int
		case reflect.String:
			fi.MWType = mw_pk_string
		default:
			panic(fmt.Sprintf("unsupported type (%s) for primary key", fi.ReflectType.Kind()))
		}
	case "", "nullable": // Do nothing special
	default:
		panic("Invalid mw tag " + tagVal)
	}

	if fi.MWName == "id" && mod.PKName == "" {
		switch fi.ReflectType.Kind() {
		case reflect.Int, reflect.Int32, reflect.Int64:
			fi.MWType = mw_pk_int
		case reflect.String:
			fi.MWType = mw_pk_string
		default:
			panic(fmt.Sprintf("unsupported type (%s) for primary key", fi.ReflectType.Kind()))
		}
		mod.PKName = "id"
	} else if fi.ReflectType.String() == timeType {
		// Slight hack to see if this is a time object, otherwise we'll use the long below switch
		fi.MWType = mw_date_time
	} else {
		fi.MWType = getMWBaseType(fi.ReflectKind)
	}
	if strings.Contains(fi.MWType, "{nullable}") {
		var nullStr string
		if !fi.Nullable {
			nullStr = "NOT NULL"
		}
		fi.MWType = strings.Replace(fi.MWType, "{nullable}", nullStr, 1)
	}
}

func getMWBaseType(goType reflect.Kind) string {
	switch goType {
	// All supported jsonb types. TODO test ptr, might have to dive
	case reflect.Array, reflect.Slice:
		return mw_json

	case reflect.Map, reflect.Struct:
		return mw_json
	// Small int
	case reflect.Int8, reflect.Int16:
		return mw_small_int
	// Note postgres does not support uint, so we'll just use int
	case reflect.Int, reflect.Int32, reflect.Uint, reflect.Uint32, reflect.Uint8,
		reflect.Uint16:
		return mw_integer
	case reflect.Int64, reflect.Uint64:
		return mw_integer
	case reflect.Float32:
		return mw_float
	case reflect.Float64:
		return mw_float
	case reflect.String:
		return mw_text
	case reflect.Bool:
		return mw_boolean
	}
	panic(fmt.Sprintf("Unsupported type %d (%s)", goType, goType))
}

func parseName(name string) string {
	buf := bytes.NewBuffer(make([]byte, 0, 2*len(name)))

	var (
		upperCount               int
		writeUnderscode, isUpper bool
		runes                    = []rune(name)
	)
	for i := range runes {
		isUpper = unicode.IsUpper(runes[i])
		writeUnderscode = i != 0 && upperCount == 0 && isUpper
		if writeUnderscode {
			buf.WriteByte('_')
		}
		if !isUpper {
			// in case there are capitalized letters before camelcase, lile
			// JSONString, so we want to split this into json_string
			if upperCount > 1 {
				for j := 0; j < upperCount-1; j++ {
					buf.WriteRune(unicode.ToLower(runes[i-upperCount+j]))
				}
				buf.WriteByte('_')
			}
			// in case previous letter is capital, write it before current lower letter
			if upperCount > 0 {
				buf.WriteRune(unicode.ToLower(runes[i-1]))
			}
			buf.WriteRune(runes[i])
			upperCount = 0
		} else {
			upperCount++
		}
	}
	// if the last part of string is capitalized, like MyStringJSON, write last part to buffer
	if isUpper {
		for j := 0; j < upperCount; j++ {
			buf.WriteRune(unicode.ToLower(runes[len(runes)-upperCount+j]))
		}
	}

	return buf.String()
}
