package main

import (
	"fmt"
	"time"

	mw "github.com/cliqueinc/mysql-wear"
)

type Email struct {
	ID              string
	EmailID         string
	RejectionReason string `mw:"nullable"`
	Sent            time.Time
	Created         time.Time
	Updated         time.Time
}

func main() {
	fmt.Println(mw.GenerateSchema(&Email{}))
	fmt.Println(mw.GenerateModel(&Email{}, "e"))
	fmt.Println(mw.GenerateModelTest(&Email{}, "e"))
}
