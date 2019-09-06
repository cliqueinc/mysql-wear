package mwear

import (
	"io/ioutil"
	"strconv"
	"strings"
)

const (
	// NAMES of variables NOT to be confused with their values
	// Compare to ConnectVals which is the final product of this file
	EnvHost          = "MYSQL_HOST"
	EnvPort          = "MYSQL_PORT"
	EnvDBName        = "MYSQL_DB_NAME"
	EnvUserName      = "MYSQL_USER_NAME"
	EnvPassword      = "MYSQL_PASSWORD"
	EnvMigrationPath = "MYSQL_MIGRATION_PATH"
	EnvUseTLS        = "MYSQL_USE_TLS"
	MaxPort          = 65535 // 2^16 - 1
)

// TODO IMPLEMENT
func ParseEnvFromOS() (*ConnectVals, error) {
	panic("Not implemented")
	return nil, nil
}

// Use a very simple .mysqlenv file
func ReadEnvFile(fileName string) (*ConnectVals, error) {
	contents, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, err
	}
	cv := parseEnv(string(contents))
	return cv, nil

}

// ParseEnv Generally passed file data from ReadEnvFile of the form:
// MYSQL_HOST=localhost
// And just ignores errors
func parseEnv(data string) *ConnectVals {
	cv := &ConnectVals{}
	for _, line := range strings.Split(data, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lineParts := strings.Split(line, "=")
		if len(lineParts) != 2 {
			continue
		}
		switch lineParts[0] {
		case EnvHost:
			cv.Host = lineParts[1]
		case EnvPort:
			cv.Port = parsePort(lineParts[1])
		case EnvDBName:
			cv.DBName = lineParts[1]
		case EnvUserName:
			cv.UserName = lineParts[1]
		case EnvPassword:
			cv.Password = lineParts[1]
		case EnvMigrationPath:
			cv.MigrationPath = lineParts[1]
		case EnvUseTLS: // Defaults to true
			cv.UseTLS = parseUseTLS(lineParts[1])
		}
	}
	return cv
}

// Return the port if valid
func parsePort(p string) int {
	pInt, err := strconv.Atoi(p)
	if err != nil {
		return 0
	}
	if pInt < 0 || pInt > MaxPort { // Greater than largest port # 65535
		return 0
	}
	return pInt
}

func parseUseTLS(ut string) bool {
	if ut == "false" {
		return false
	}
	return true
}
