package mwear

import (
	"fmt"
	"os"
	"strings"
)

func PathExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	// We got the error we were expecting, DNE
	if os.IsNotExist(err) {
		return false
	}
	panic(fmt.Sprintf("Unexpected error (%v) checking if path (%s) exists", err, path))
}

// GetColumnName returns the column name with prepending table name.
func GetColumnName(modPtr interface{}, colName string) string {
	trimString := func(str string) string {
		return strings.Replace(strings.Replace(str, ";", "", -1), "`", "", -1)
	}
	escapeString := func(str string) string {
		return "`" + trimString(str) + "`"
	}
	modInfo := parseModel(modPtr, false)

	return "`" + modInfo.TableName + "`." + escapeString(colName)
}

// MakeOrderBy makes "order by" sql clause by parsing incoming field string and model type.
// Field string is expected to be either struct field name, or underscored pg column name,
// For "userID" field name the output will be "order by user_id asc",
// for the field with "-" prefix, like "-userID", the output is "order by user_id desc".
// if there is no userID (or UserID, userid etc) field in struct, or no user_id column name,
// empty string returned.
func MakeOrderBy(modelPtr interface{}, sortBy string) (orderField, direction string, ok bool) {
	sortBy = strings.TrimSpace(sortBy)
	if sortBy == "" {
		return "", "", false
	}

	direction = "ASC"
	if strings.HasPrefix(sortBy, "-") {
		direction = "DESC"
		sortBy = sortBy[len("-"):]
	}

	var (
		mod         = parseModel(modelPtr, false)
		fieldExists bool
	)
	for _, f := range mod.Fields {
		if f.MWName == sortBy || strings.ToLower(f.GoName) == strings.ToLower(sortBy) {
			fieldExists = true
			sortBy = f.MWName
			break
		}
	}
	if !fieldExists {
		return "", "", false
	}

	return sortBy, direction, true
}
