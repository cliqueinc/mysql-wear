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
	{{- if eq $i (minus (len $.Fields) 1) }}{{$e.MWNameQuoted}} {{$e.MWType}}
	{{- else -}} {{$e.MWNameQuoted}} {{$e.MWType}},
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

func Get{{.StructName}}(db *mw.DB, id {{ .GetPKField.ReflectType.String }}) (*{{.StructName}}, error) {
	{{.ShortName}} := &{{.StructName}}{ID: id}
	found, err := db.Get({{.ShortName}})
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}
	return {{.ShortName}}, nil
}

func ({{.ShortName}} *{{.StructName}}) Insert(db *mw.DB) error {
	{{.ShortName}}.Created = time.Now().UTC()
	{{.ShortName}}.Updated = time.Now().UTC()
	res, err := db.Insert({{.ShortName}})
	if err != nil {
		return err
	}
	{{ if .IsIntPK -}}
	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("fail get last id: %v", err)
	}
	{{ if eq .GetPKKind 2 -}}
	{{.ShortName}}.ID = int(id)
	{{ else if eq .GetPKKind 5 -}}
	{{.ShortName}}.ID = int32(id)
	{{ else -}}
	{{.ShortName}}.ID = id
	{{ end -}}
	{{ end -}}
	return nil
}

func ({{.ShortName}} *{{.StructName}}) Update(db *mw.DB) error {
	{{.ShortName}}.Updated = time.Now().UTC()
	if err := db.Update({{.ShortName}}); err != nil {
		return err
	}
	return nil
}

func ({{.ShortName}} *{{.StructName}}) Delete(db *mw.DB) error {
	if err := db.Delete({{.ShortName}}); err != nil {
		return err
	}
	return nil
}

`

const modelTestTemplate = `
// -------------------------------------------- //
// AUTO GENERATED - Place in a new model_test file
// -------------------------------------------- //

import (
	"testing"
	mw "github.com/cliqueinc/mysql-wear"
)

var db *mw.DB

func init() {
	// suppose you have some helper code for initializing db connection
	db = dbtest.InitDB()
}

func Test{{.StructName}}CRUD(t *testing.T){
	{{.ShortName}} := New{{.StructName}}()
	// Fill in struct properties here, especially the ID/PK field


	if err := {{.ShortName}}.Insert(db); err != nil {
		log.Fatalf("insert failed: %v", err)
	}

	// Make sure we can get the newly inserted object
	{{.ShortName}}2, err := Get{{.StructName}}(db, {{.ShortName}}.ID)
	if err != nil {
		log.Fatalf("fail get item {{ if .IsIntPK }}%d{{else}}%s{{end}}: %v", {{.ShortName}}.ID, err)
	}
	if {{.ShortName}}2 == nil {
		t.Fatalf("Didnt find newly inserted row with ID {{ if .IsIntPK }}%d{{else}}%s{{end}}", {{.ShortName}}.ID)
	}
	// Make some changes to {{.ShortName}} here


	if err := {{.ShortName}}.Update(db); err != nil {
		log.Fatalf("id ({{ if .IsIntPK }}%d{{else}}%s{{end}}): update failed: %v", {{.ShortName}}.ID, err)
	}

	// Make sure those changes took effect
	{{.ShortName}}3, err := Get{{.StructName}}(db, {{.ShortName}}.ID)
	if err != nil {
		log.Fatalf("fail get item {{ if .IsIntPK }}%d{{else}}%s{{end}}: %v", {{.ShortName}}3.ID, err)
	}
	if {{.ShortName}}3 == nil {
		t.Fatalf("Missing row 3 ID {{ if .IsIntPK }}%d{{else}}%s{{end}}", {{.ShortName}}3.ID)
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
