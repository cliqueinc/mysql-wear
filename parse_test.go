package mwear

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/cliqueinc/mysql-wear/util"
)

// See ops_test for some good test structs
func TestParseModelSimple(t *testing.T) {

	// TODO might be able to turn these into table tests
	type simpleAddress struct {
		Street string
		State  string
		City   string
		ID     string
	}

	addr := &simpleAddress{
		fmt.Sprintf("%d %s %s", util.RandomInt(10000, 999999),
			util.RandomString(10), util.RandomString(10)),
		util.RandomString(2),
		util.RandomString(8) + " " + util.RandomString(5),
		util.RandomString(40),
	}

	model := parseModel(addr, true)
	if model == nil {
		t.Fatalf("parseModel failed, got nil model back")
	}
	if model.StructName != "simpleAddress" || model.ReflectType.Kind() != reflect.Ptr {
		t.Errorf("parseModel failed, expected name simpleAddress type ptr, %v", model)
	}
	if model.TableName != "simple_address" {
		t.Errorf("Expected TableName simple_address, got %s", model.TableName)
	}

	if len(model.Fields) < 3 {
		t.Fatalf("Didnt find three fields, found %d", len(model.Fields))
	}

	// This is a fairly stupid test, ug. Probably change to table tests
	for i, f := range model.Fields {
		switch i {
		case 0:
			if f.GoName != "Street" || f.MWName != "street" {
				t.Errorf("Expected Go:Street, PG:street, was %s %s", f.GoName, f.MWName)
			}
		case 1:
			if f.GoName != "State" || f.MWName != "state" {
				t.Errorf("Expected Go:State, PG:state, was %s %s", f.GoName, f.MWName)
			}
		case 2:
			if f.GoName != "City" || f.MWName != "city" {
				t.Errorf("Expected Go:City, PG:city, was %s %s", f.GoName, f.MWName)
			}
		}
		if !strings.HasPrefix(f.MWType, "VARCHAR") { // Could be text with PRIMARY KEY
			t.Errorf("Expected pg type text, got %s", f.MWType)
		}
		if f.ReflectKind != reflect.String {
			t.Errorf("Expected reflect go type king string, got %d", f.ReflectKind)
		}
	}
}

func assertPanicParseModel(t *testing.T, badThing interface{}) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("assertPanicParseModel should have panicked")
		}
	}()
	parseModel(badThing, true)
}

// This test should panic on the current not supported struct pointer field
func TestParseModelPanicPtrField(t *testing.T) {
	type ptrAddress struct {
		Street string
		State  *string
		City   string
	}
	assertPanicParseModel(t, &ptrAddress{})
}

// This test should panic on the current not supported struct pointer field
func TestParseModelPanicNonStruct(t *testing.T) {
	type ptrAddress struct {
		Street string
		State  *string
		City   string
	}
	assertPanicParseModel(t, ptrAddress{})
	assertPanicParseModel(t, 42)
}

type tableMeth struct {
	ID string
}

func (t tableMeth) TableName() string {
	return "not_like_the_real_meth"
}

func TestGetTableName(t *testing.T) {
	// Going to avoid using parseModel() directly for this test to isolate things

	tm := &tableMeth{ID: util.RandomString(30)}

	mod := &model{
		Struct:      tm,
		ReflectType: reflect.TypeOf(tm),
	}
	rowModel := reflect.ValueOf(tm)
	mod.StructName = rowModel.Elem().Type().Name()
	mod.setTableName(rowModel)
	if mod.TableName != "not_like_the_real_meth" {
		t.Errorf("Expected (%s), was (%s)", "not_like_the_real_meth", mod.TableName)
	}
}

func TestGetTableNameNoMeth(t *testing.T) {
	// Going to avoid using parseModel() directly for this test to isolate things
	type noTableMeth struct{ ID string }
	tm := &noTableMeth{ID: util.RandomString(30)}

	mod := &model{
		Struct:      tm,
		ReflectType: reflect.TypeOf(tm),
	}
	rowModel := reflect.ValueOf(tm)
	mod.StructName = rowModel.Elem().Type().Name()
	mod.setTableName(rowModel)
	if mod.TableName != "no_table_meth" {
		t.Errorf("Expected (%s), was (%s)", "no_table_meth", mod.TableName)
	}
}

func Test_ParseName(t *testing.T) {
	var cases = []struct {
		Name          string
		Input, Output string
	}{
		{
			Name:   "simple camelcase",
			Input:  "MyCamelCasedName",
			Output: "my_camel_cased_name",
		},
		{
			Name:   "id",
			Input:  "ID",
			Output: "id",
		},
		{
			Name:   "all letters capitalized",
			Input:  "URL",
			Output: "url",
		},
		{
			Name:   "capitalized after camelcase",
			Input:  "MyJSONString",
			Output: "my_json_string",
		},
		{
			Name:   "ends with capitalized",
			Input:  "MyStringJSON",
			Output: "my_string_json",
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			o := parseName(c.Input)
			if o != c.Output {
				t.Errorf("parseName failed: expected (%s), got (%s)", c.Output, o)
			}
		})
	}
}

func Benchmark_ParseName(b *testing.B) {
	names := []string{
		"MyBigCamelCasedStringThatIsVeryLong",
		"SimpleString",
		"under_scored",
	}
	var rep = regexp.MustCompile(`[A-Z]+`)
	oldParseFunc := func(name string) string {
		if name == "ID" {
			// Primary key
			return "id"
		}

		for true {
			if name == strings.ToLower(name) {
				return name
			}
			name = rep.ReplaceAllStringFunc(name, func(s string) string {
				return "_" + strings.ToLower(s)
			})
			name = strings.TrimLeft(name, "_")
		}
		panic("unreachable")
	}

	b.ReportAllocs()
	b.Run("v1", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, name := range names {
				oldParseFunc(name)
			}
		}
	})
	b.Run("v2", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, name := range names {
				parseName(name)
			}
		}
	})
}
