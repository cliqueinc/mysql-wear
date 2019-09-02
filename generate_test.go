package mwear_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	mw "github.com/cliqueinc/mysql-wear"
)

// Example email struct used for generating sql, model, test
type Email struct {
	ID              string
	EmailID         string
	RejectionReason string `mw:nullable`
	Sent            time.Time
	Created         time.Time
	Updated         time.Time
}

func ExampleGenerateModel() {
	fmt.Println(mw.GenerateSchema(&Email{}))
	fmt.Println(mw.GenerateModel(&Email{}, "email"))
	fmt.Println(mw.GenerateModelTest(&Email{}, "email"))
}

func assertContains(t *testing.T, str string, substr string) {
	if !strings.Contains(str, substr) {
		t.Errorf("Got: %v which doesn't contain: %v", str, substr)
	}
}

func TestGenerateSchema(ts *testing.T) {
	ts.Run("Int primary key", func(t *testing.T) {
		type UserProfile1 struct {
			ID int `mw:"pk"`
		}
		schema := mw.GenerateSchema(&UserProfile1{})
		assertContains(t, schema, "`id` INT NOT NULL AUTO_INCREMENT PRIMARY KEY")
	})

	ts.Run("Primary key with custom field name", func(t *testing.T) {
		type UserProfile2 struct {
			PK int `mw:"pk"`
		}
		schema := mw.GenerateSchema(&UserProfile2{})
		assertContains(t, schema, "`pk` INT NOT NULL AUTO_INCREMENT PRIMARY KEY")
	})

	ts.Run("Primary key without tag", func(t *testing.T) {
		type UserProfile3 struct {
			ID int
		}
		schema := mw.GenerateSchema(&UserProfile3{})
		assertContains(t, schema, "`id` INT NOT NULL AUTO_INCREMENT PRIMARY KEY")
	})

	ts.Run("2", func(t *testing.T) {
		type UserProfile4 struct {
			ID string `mw:"pk"`
		}
		schema := mw.GenerateSchema(&UserProfile4{})
		assertContains(t, schema, "`id` VARCHAR(255) NOT NULL PRIMARY KEY")
	})
}
