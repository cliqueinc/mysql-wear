package mwear

import (
	"testing"
	"time"
)

func TestAdapter(t *testing.T) {
	type fakeProfile struct {
		Bio     string
		PicUrl  string
		Meta    map[string]interface{}
		SubTime time.Time
	}

	type fakeAddress struct {
		Street string
		State  string
		City   string
	}

	type weird struct {
		Name            string
		OptionalProfile *fakeProfile
	}

	type fakeTransactionUser struct {
		ID        string
		FirstName string
		LastName  string
		Emails    []string
		Addresses []fakeAddress
		Profile   fakeProfile
		Meta      map[string]interface{}
		Created   time.Time
	}

	fu := &fakeTransactionUser{
		ID:        RandomString(30),
		FirstName: "Jym",
		LastName:  "Luast",
		Emails:    []string{"but@butts.com", "second@whowhooo.com"},
		Addresses: []fakeAddress{
			{"750 N. San Vicente", "CA", "LA"},
			{"8800 Sunset BLVD", "CA", "LA"},
		},
		Profile: fakeProfile{"Some Lame bio", "http://picsrus.com/23ds34", map[string]interface{}{
			"facebookid": "123k1l2j34l13", "twitterid": "sdklj324lkj23"}, time.Now()},
		Meta: map[string]interface{}{
			"favcolor": "brown", "fakeage": 42,
		},
	}
	fu2 := &fakeTransactionUser{
		ID:        RandomString(30),
		FirstName: "John",
		LastName:  "Snow",
		Emails:    []string{"but@butts.com", "second@whowhooo.com"},
		Addresses: []fakeAddress{
			{"750 N. San Vicente", "CA", "LA"},
			{"8800 Sunset BLVD", "CA", "LA"},
		},
		Profile: fakeProfile{"Some Lame bio", "http://picsrus.com/23ds34", map[string]interface{}{
			"facebookid": "123k1l2j34l13", "twitterid": "sdklj324lkj23"}, time.Now()},
		Meta: map[string]interface{}{
			"favcolor": "brown", "fakeage": 42,
		},
	}

	t.Run("CRUD", func(t *testing.T) {
		type adapterUser struct {
			ID   string
			Name string
		}

		u := &adapterUser{
			ID:   RandomString(30),
			Name: RandomString(10),
		}

		db.MustCreateTable(u)
		if _, err := db.Insert(u); err != nil {
			t.Fatalf("failed to insert row: %s", err)
		}
	})

	if err := db.CreateTable(fu); err != nil {
		t.Fatalf("failed to create schema: %s", err)
	}

	t.Run("Rollback", func(t *testing.T) {
		tx, err := db.Begin()
		if err != nil {
			t.Fatalf("cannot begin transaction: %s", err)
		}

		a := Wrap(tx)
		res, err := a.Insert(fu)
		if err != nil {
			t.Fatalf("failed to insert row: %s", err)
		}
		num, err := res.RowsAffected()
		if err != nil {
			t.Fatalf("%v", err)
		}
		t.Log(num)
		_, err = a.Insert(fu2)
		if err != nil {
			t.Fatalf("failed to insert row: %s", err)
		}

		fu.FirstName = RandomString(30)
		err = a.Update(fu)
		if err != nil {
			t.Fatalf("failed to update row: %s", err)
		}

		err = a.Delete(fu2)
		if err != nil {
			t.Fatalf("failed to delete row: %s", err)
		}

		err = tx.Rollback()
		if err != nil {
			t.Fatalf("failed to rollback transaction: %s", err)
		}

		fu3 := &fakeTransactionUser{ID: fu.ID}
		found, err := db.Get(fu3)
		if found {
			t.Error("transaction changes should have been discarded")
		}
	})

	t.Run("Commit", func(t *testing.T) {
		tx, err := db.Begin()
		if err != nil {
			t.Fatalf("cannot begin transaction: %s", err)
		}
		a := Wrap(tx)

		res, err := a.Insert(fu)
		if err != nil {
			t.Fatalf("failed to insert row: %s", err)
		}
		num, err := res.RowsAffected()
		if err != nil {
			t.Fatalf("%v", err)
		}
		t.Log(fu.ID, num)

		err = tx.Commit()
		if err != nil {
			t.Fatalf("failed to commit transaction: %s", err)
		}

		fu3 := &fakeTransactionUser{ID: fu.ID}
		found, err := db.Get(fu3)
		if err != nil {
			t.Fatalf("failed get user (%s): %s", fu.ID, err)
		}
		t.Log(found)
		if !found || fu3.FirstName != fu.FirstName {
			t.Error("transaction changes should have been preserved")
		}
	})
}
