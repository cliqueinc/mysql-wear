package mwear

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/go-sql-driver/mysql"
)

func IsUniqueViolationError(err error) bool {
	if err == nil {
		return false
	}
	if mysqlError, ok := err.(*mysql.MySQLError); ok {
		if mysqlError.Number == 1062 { // duplicate entry error code for mysql
			return true
		}
	}

	return false
}

// IsTableExistsError checks whether an error is table already exists error.
func IsTableExistsError(err error) bool {
	if err == nil {
		return false
	}
	if mysqlError, ok := err.(*mysql.MySQLError); ok {
		if mysqlError.Number == 1050 { // table exists error code for mysql
			return true
		}
	}

	return false
}

type jsonScanner struct {
	item interface{}
}

func (scanner *jsonScanner) Scan(val interface{}) error {
	if val == nil {
		return nil
	}
	switch v := val.(type) {
	case []byte:
		json.Unmarshal(v, &scanner.item)
		return nil
	case string:
		json.Unmarshal([]byte(v), &scanner.item)
		return nil
	default:
		return fmt.Errorf("unsupported type: %T", v)
	}
}

func (a *Adapter) rawSelect(sqlStmt string, columns []string, joinMods []*model, joinFields [][]*field, requirePK bool, sliceValElement reflect.Value,
	sliceTypeElement reflect.Type, args ...interface{}) error {

	if debugEnabled {
		fmt.Println(sqlStmt, args)
	}

	rows, err := a.con.Query(sqlStmt, args...)
	if err != nil {
		return err
	}

	var prevRow struct {
		modPK            string
		model            reflect.Value
		parsedJoinModels map[string][]string
	}
	var (
		mod       *model
		modFields []*field
		modPK     string
		valAddrs  []interface{}
		rowModel  reflect.Value
		rowJoins  []reflect.Value
	)

	defer rows.Close()
	for rows.Next() {
		if mod == nil {
			mod = parseModel(reflect.New(sliceTypeElement).Interface(), requirePK)
			modFields = mod.getFields(columns)
			valAddrs = make([]interface{}, 0, len(modFields))
		} else {
			valAddrs = valAddrs[:0]
			rowJoins = rowJoins[:0]
		}
		rowModel = reflect.New(mod.ReflectType.Elem())
		for i := range joinMods {
			rowJoins = append(rowJoins, reflect.New(joinMods[i].ReflectType.Elem()))
		}
		for i := range modFields {
			val := rowModel.Elem().Field(modFields[i].FieldPos).Addr().Interface()
			if modFields[i].MWType == mw_json {
				val = &jsonScanner{val}
			} else if modFields[i].Nullable {
				val = &nullScanner{rowModel.Elem().Field(modFields[i].FieldPos), modFields[i]}
			}
			valAddrs = append(valAddrs, val)
		}
		for i := range joinMods {
			for ind := range joinFields[i] {
				fv := rowJoins[i].Elem().Field(joinFields[i][ind].FieldPos)
				scanner := &nullScanner{fv, joinFields[i][ind]}
				valAddrs = append(valAddrs, scanner)
			}
		}

		err = rows.Scan(valAddrs...)
		if err != nil {
			panic(err)
		}

		if mod.PKPos != -1 {
			modPK = mod.getPK(rowModel)
		}
		// rowIsTheSame is used for checks whether the next row represents the same
		// model, and if yes it means that the difference is in the different join model,
		// which happens in one to many relation.
		rowIsTheSame := modPK != "" && modPK == prevRow.modPK
		if len(joinMods) != 0 {
			for i := range joinMods {
				joinName := joinMods[i].ReflectType.Elem().Name()
				joinPos, ok := mod.Joins[joinName]
				if !ok {
					panic(fmt.Sprintf("unknown join %s", joinMods[i].ReflectType.Elem().Name()))
				}
				modJoin := rowModel.Elem().Field(joinPos)

				var joinPKVal string
				if joinMods[i].PKPos != -1 {
					joinPKVal = joinMods[i].getPK(rowJoins[i])
				}

				// during join select we replace possible joined null values with default values,
				// as pgx don't want to parse null into string), so we just check whether
				// joined primary key is empty, which means that this row don't have anything joined.
				if joinPKVal == "" {
					continue
				}

				// if case its one to one join
				if modJoin.Kind() != reflect.Slice {
					if modJoin.Kind() == reflect.Ptr {
						modJoin.Set(rowJoins[i])
					} else {
						modJoin.Set(rowJoins[i].Elem())
					}
					continue
				}

				// in case one-to-many join we want to ensure that we haven't already added this
				// join to our model, thats why we keep added models in prevRow.parsedJoinModels map.
				var modelAlreadySet bool
				if prevRow.parsedJoinModels != nil {
					if parsedModels, ok := prevRow.parsedJoinModels[joinName]; ok {
						for _, mID := range parsedModels {
							if mID == joinPKVal {
								modelAlreadySet = true
								break
							}
						}
					}
				} else {
					prevRow.parsedJoinModels = make(map[string][]string)
				}
				if modelAlreadySet {
					continue
				}
				prevRow.parsedJoinModels[joinName] = append(prevRow.parsedJoinModels[joinName], joinPKVal)

				// set current join model to our real model.
				if rowIsTheSame {
					prevVal := prevRow.model.Elem().Field(joinPos)
					prevVal.Set(reflect.Append(prevVal, rowJoins[i].Elem()))
				} else {
					slice := reflect.MakeSlice(reflect.SliceOf(joinMods[i].ReflectType.Elem()), 0, 1)
					rowModel.Elem().Field(joinPos).Set(reflect.Append(slice, rowJoins[i].Elem()))
				}
			}
		}
		if !rowIsTheSame {
			prevRow.model = rowModel
			prevRow.modPK = modPK
		}

		if !rowIsTheSame {
			// if our model is scanned first time, just append it to other models
			sliceValElement.Set(reflect.Append(sliceValElement, rowModel.Elem()))
		} else {
			// if this model was already scanned, but some joins were added to it,
			// we just want to update the latest slice element with newest changes.
			sliceValElement.Index(sliceValElement.Len() - 1).Set(prevRow.model.Elem())
		}
	}
	err = rows.Err()

	if err != nil {
		return err
	}

	return nil
}
