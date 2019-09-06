package mwear

import (
	"io/ioutil"
	"math"
	"os"
	"reflect"
	"testing"
)

func TestParseEnv(t *testing.T) {

	goodEnvFile := `
	MYSQL_HOST=localhost
	MYSQL_PORT=1323
	MYSQL_DB_NAME=hydra
	MYSQL_USER_NAME=bill
	MYSQL_PASSWORD=bill1zawesome
	MYSQL_MIGRATION_PATH=/abs/paath
	MYSQL_USE_TLS=true
	`
	goodCv := &ConnectVals{
		Host:          "localhost",
		Port:          1323,
		DBName:        "hydra",
		UserName:      "bill",
		Password:      "bill1zawesome",
		MigrationPath: "/abs/paath",
		UseTLS:        true,
	}
	if !reflect.DeepEqual(*parseEnv(goodEnvFile), *goodCv) {
		t.Error("Structs didnt match")
	}
	badEnvFile := `
	MYSQL_HOST=localhoster
	***xxYzzzz23lkj
	MYSQL_DB_NAME=hydra
	ggffyzz=bdlmnop
	MYSQL_PASSWORD=bill1zawesome
	MYSQL_MIGRATION_PATH=/abs/paath
	MYSQL_USE_TLS=false
	`
	badCv := &ConnectVals{
		Host:          "localhoster",
		Port:          0,
		DBName:        "hydra",
		UserName:      "",
		Password:      "bill1zawesome",
		MigrationPath: "/abs/paath",
		UseTLS:        false,
	}
	if !reflect.DeepEqual(*parseEnv(badEnvFile), *badCv) {
		t.Error("Structs didnt match")
	}
}

func TestReadEnvFile(t *testing.T) {
	goodEnvFile := `
	MYSQL_HOST=localhost
	MYSQL_PORT=1323
	MYSQL_DB_NAME=hydra
	MYSQL_USER_NAME=bill
	MYSQL_PASSWORD=bill1zawesome
	MYSQL_MIGRATION_PATH=/abs/paath
	MYSQL_USE_TLS=true
	`
	goodCv := &ConnectVals{
		Host:          "localhost",
		Port:          1323,
		DBName:        "hydra",
		UserName:      "bill",
		Password:      "bill1zawesome",
		MigrationPath: "/abs/paath",
		UseTLS:        true,
	}
	tmpFile, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatalf("Failed to create temp file %s", err)
	}
	fileName := tmpFile.Name()
	t.Logf("Created temp file %s", fileName)
	_, err = tmpFile.WriteString(goodEnvFile)
	if err != nil {
		t.Fatalf("Failed to write to temp file %s", err)
	}
	err = tmpFile.Close()
	if err != nil {
		t.Fatalf("Failed to close temp file %s", err)
	}
	cv, err := ReadEnvFile(fileName)
	if err != nil {
		t.Fatalf("ReadEnvFile Failed %s", err)
	}
	if !reflect.DeepEqual(*cv, *goodCv) {
		t.Error("Structs didnt match")
	}
	err = os.Remove(fileName)
	if err != nil {
		t.Errorf("Failed to remove temp file %s", err)
	}
}

func TestParsePort(t *testing.T) {
	type testCase struct {
		s         string
		expectedI int
	}
	cases := []testCase{
		{"10", 10},
		{"-1", 0},
		{"X", 0},
		{"abcd10xyz", 0},
		{"65535", 65535},
		{"65536", 0},
		{"1333", 1333},
	}
	for _, c := range cases {
		if res := parsePort(c.s); res != c.expectedI {
			t.Errorf("(%d) Failed on (%s), expected (%d) got (%d)", int(math.Pow(2, 16)), c.s, c.expectedI, res)
		}
	}
}
