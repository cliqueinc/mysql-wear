package mwear_test

import (
	"fmt"
	"time"

	mw "github.com/cliqueinc/mysql-wear"
)

// Example email struct used for generating sql, model, test
type Email struct {
	ID              string
	EmailID         string
	RejectionReason string
	Sent            time.Time
	Created         time.Time
	Updated         time.Time
}

func ExampleGenerateModel() {
	fmt.Println(mw.GenerateSchema(&Email{}))
	fmt.Println(mw.GenerateModel(&Email{}, "email"))
	fmt.Println(mw.GenerateModelTest(&Email{}, "email"))
}
