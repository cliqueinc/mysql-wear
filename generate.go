package mwear

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"sync"
	"text/template"
)

var (
	tmpls map[string]*template.Template
	mx    sync.Mutex
)

func renderTemplate(mod interface{}, sqlTemplate string) string {
	buff := &bytes.Buffer{}

	hasher := md5.New()
	hasher.Write([]byte(sqlTemplate))
	hash := hex.EncodeToString(hasher.Sum(nil))
	if tmpls == nil {
		tmpls = make(map[string]*template.Template)
	}
	var (
		tmpl *template.Template
		err  error
	)
	if existing, ok := tmpls[hash]; ok {
		tmpl = existing
	} else {
		tmpl, err = template.New("sql").Funcs(funcMap).Parse(sqlTemplate)
		if err != nil {
			panic(err)
		}
		mx.Lock()
		tmpls[hash] = tmpl
		mx.Unlock()
	}

	err = tmpl.Execute(buff, mod)
	if err != nil {
		panic(err)
	}
	return buff.String()
}

const insertTemplate = `
INSERT INTO ` + "`{{.model.TableName}}`" + `(
	{{ range $i, $e := .model.Fields }}
	{{- if eq $i (minus (len $.model.Fields) 1) }}{{$e.MWNameQuoted}}
	{{- else -}} {{$e.MWNameQuoted}},
	{{end -}}
{{- end }}
) VALUES 
	{{- range $itemNum, $item := .Items }}(
		{{ range $i, $e := $item.Fields -}}
		?{{- if ne $i (minus (len $.model.Fields) 1) }},{{end -}}
		{{end }}
	){{- if ne $itemNum (minus (len $.Items) 1) }},{{- end }}
	{{end -}}
;
`
const selectBaseTemplate = `SELECT
	{{ range $i, $e := .fields }}
	{{- if eq $i (minus (len $.fields) 1) }}{{$e.MWNameQuotedSelect}}
	{{- else -}} {{$e.MWNameQuotedSelect}},
	{{end -}}
	{{end }}
	{{- range $joinInd, $joinMod := .joinMods }}
		, 
		{{ range $i, $e := index $.joinFields $joinInd  }}
			{{- if eq $i (minus (len (index $.joinFields $joinInd)) 1) }}{{$e.JoinedMWName}}
			{{- else -}} {{$e.JoinedMWName}},
		{{end -}}
		{{end }}
	{{end }}
	FROM ` + "`{{.mod.TableName}}`" + ` 
	{{ range $jcfg := .joins }}LEFT JOIN {{$jcfg.TableName}} ON {{$jcfg.Condition}} {{end }} `

const queryByPKTemplate = `WHERE "{{.PKName}}" = '{{.PKValue}}'
`

// TODO use a variable for the count for $1 $2... since we cant use the index due to
// removing the id field
const updateTemplate = `
UPDATE ` + "`{{.mod.TableName}}`" + ` SET
	{{ range $i, $e := .fields }}
	{{- if eq $i (minus (len $.fields) 1) }}{{$e.MWNameQuoted}} = ?
	{{- else -}}{{$e.MWNameQuoted}} = ?,
	{{end -}}
{{- end }}
`

const deleteTemplate = "DELETE FROM `{{.TableName}}`"

var funcMap = template.FuncMap{
	"minus": minus,
	"plus":  plus,
	"mul":   multiply,
}

func minus(a, b int) int {
	return a - b
}

func plus(nums ...int) int {
	var sum int
	for _, n := range nums {
		sum += n
	}
	return sum
}

func multiply(nums ...int) int {
	if len(nums) == 0 {
		return 0
	}

	total := 1
	for _, n := range nums {
		total = total * n
	}
	return total
}

// Note this is pretty ugly to deal with whitespace issues
// The if inside the range checks if we are on the last iteration to omit comma
const createTableTemplate = `
-- AUTO GENERATED - place in a new schema migration <#>/up.sql

CREATE TABLE ` + "`{{.TableName}}`" + `(
	{{ range $i, $e := .Fields }}
	{{- if eq $i (minus (len $.Fields) 1) }}{{$e.MWName}} {{$e.MWType}}
	{{- else -}} {{$e.MWName}} {{$e.MWType}},
	{{end -}}
{{- end }}
);
`

const modelTemplate = `
// -------------------------------------------- //
// AUTO GENERATED - Place in a new models file
// -------------------------------------------- //

func New{{.StructName}}() *{{.StructName}}{
	return &{{.StructName}}{}
}

func Get{{.StructName}}(id string) *{{.StructName}}{
	{{.ShortName}} := &{{.StructName}}{ID: id}
	pgc.Get({{.ShortName}})
	return {{.ShortName}}
}

func ({{.ShortName}} *{{.StructName}}) Insert(){
	{{.ShortName}}.Created = time.Now().UTC()
	{{.ShortName}}.Updated = time.Now().UTC()
	pgc.Insert({{.ShortName}})
}

func ({{.ShortName}} *{{.StructName}}) Update(){
	{{.ShortName}}.Updated = time.Now().UTC()
	pgc.Update({{.ShortName}})
}

func ({{.ShortName}} *{{.StructName}}) Delete(){
	pgc.Delete({{.ShortName}})
}

`

const modelTestTemplate = `
// -------------------------------------------- //
// AUTO GENERATED - Place in a new model_test file
// -------------------------------------------- //

import (
	"testing"
)

func init() {
	if !pgc.GetConfig().Initialized {
		pgc.InitFromEnv()
	}
}

func Test{{.StructName}}CRUD(t *testing.T){
	{{.ShortName}} := New{{.StructName}}()
	// Fill in struct properties here, especially the ID/PK field


	{{.ShortName}}.Insert()

	// Make sure we can get the newly inserted object
	{{.ShortName}}2 := Get{{.StructName}}({{.ShortName}}.ID)

	if {{.ShortName}}2 == nil {
		t.Fatalf("Didnt find newly inserted row with ID %s", {{.ShortName}}.ID)
	}
	// Make some changes to {{.ShortName}} here


	{{.ShortName}}.Update()

	// Make sure those changes took effect
	{{.ShortName}}3 := Get{{.StructName}}({{.ShortName}}.ID)
	if {{.ShortName}}3 == nil {
		t.Fatalf("Missing row 3 ID %s", {{.ShortName}}3.ID)
	}

	// Compare props

}
`

const initTemplate = `
// -------------------------------------------- //
// AUTO GENERATED - Place in a temporary go file
// -------------------------------------------- //

package main

import (
	"fmt"
	mw "github.com/cliqueinc/mysql-wear"
)

type {{.StructName}} struct {
	ID          string
	Name        string
	Description string
}

func main() {
	fmt.Println(mw.GenerateSchema(&{{.StructName}}{}))
	fmt.Println(mw.GenerateModel(&{{.StructName}}{}, "{{.ShortName}}"))
	fmt.Println(mw.GenerateModelTest(&{{.StructName}}{}, "{{.ShortName}}"))
}
`

// ------------------------------------------------------------------------- //
// Generate functions
// ------------------------------------------------------------------------- //

// This will be (probably) only used by our pgccmd to create this stub for doing
// other generation. There is not an easy way to dynamically generate things
// from structs like in other languages
func GenerateInit(structName, shortName string) string {
	return renderTemplate(map[string]string{"StructName": structName,
		"ShortName": shortName}, initTemplate)
}

// Get the create SQL statement which is generally the most useful since we need to
// add this to a schema migration file.
func GenerateModel(structPtr interface{}, shortName string) string {
	mod := parseModel(structPtr, true)
	mod.ShortName = shortName
	return renderTemplate(mod, modelTemplate)
}

func GenerateModelTest(structPtr interface{}, shortName string) string {
	mod := parseModel(structPtr, true)
	mod.ShortName = shortName
	return renderTemplate(mod, modelTestTemplate)
}

// GenerateSchema generates table schema from struct model.
func GenerateSchema(structPtr interface{}) string {
	mod := parseModel(structPtr, true)
	return renderTemplate(mod, createTableTemplate)
}
