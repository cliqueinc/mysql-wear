package mwear

import (
	"math/rand"
	"testing"
	"time"

	"github.com/cliqueinc/mysql-wear/sqlq"
	"github.com/cliqueinc/pgc/util"
)

// Please see testing guidelines in the readme

func TestFullCRUD(t *testing.T) {
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

	type fakeUser struct {
		ID        string
		FirstName string
		LastName  string
		Emails    []string
		Addresses []fakeAddress
		Profile   fakeProfile
		Meta      map[string]interface{}
		Created   time.Time
		Enabled   bool
	}

	fu := &fakeUser{
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
		Enabled: true,
	}
	db.MustCreateTable(fu)
	db.MustInsert(fu)

	fu.FirstName = RandomString(30)
	db.MustUpdate(fu)

	fu2 := &fakeUser{ID: fu.ID}
	db.MustGet(fu2)

	if fu2.FirstName != fu.FirstName {
		t.Errorf("FirstName wasnt changed after update, expected (%s) was (%s)",
			fu2.FirstName, fu.FirstName)
	}
	if !fu2.Enabled {
		t.Error("enabled expected to be true")
	}
}

type selectTest struct {
	ID      string
	Name    string
	Color   string
	Height  int
	Created time.Time
}

func TestMustSelectPanicNoSlicePtr(t *testing.T) {
	db.MustCreateTable(&selectTest{})
	// Panics because you should be passing pointer to slice not slice
	var sl []selectTest
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("TestSelectAllWherePanicNoSlicePtr should have panicked")
			}
		}()
		db.MustSelect(sl)
	}()
}

func TestSelectColumns(t *testing.T) {
	s := &selectTest{}

	s.ID = RandomString(10)
	s.Name = RandomString(10)
	s.Color = RandomString(10)
	s.Height = RandomInt(1, 5000000)
	s.Created, _ = time.Parse("2006-01-02", "2006-01-02")
	db.MustInsert(s)

	t.Run("unknown column", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("panic expected in case some column isn't recognized")
			}
		}()

		var sl []selectTest
		err := db.Select(&sl, sqlq.Columns("some column", "id"), sqlq.Limit(2))
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	t.Run("custom columns", func(t *testing.T) {
		var sl []selectTest
		err := db.Select(&sl, sqlq.Columns("id", "name"), sqlq.Limit(2))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(sl) == 0 {
			t.Fatalf("empty result, at least 1 item expected")
		}
		el := sl[0]
		if el.ID == "" || el.Name == "" {
			t.Errorf("specified columns (id,name) are empty")
		}
		if el.Color != "" || el.Height != 0 {
			t.Errorf("all columns except (id,name) should be empty, actual result: %+v", el)
		}
	})
}

func TestGet(t *testing.T) {
	type fakeGet struct {
		ID     string
		UserID string
		Name   string
	}
	f1 := &fakeGet{ID: RandomString(25), UserID: RandomString(25), Name: "BoB"}
	f2 := &fakeGet{ID: RandomString(25), UserID: RandomString(25), Name: "John"}
	db.MustCreateTable(f1)
	db.MustInsert(f1, f2)

	f1Get := &fakeGet{ID: f1.ID}
	found := db.MustGet(f1Get)
	if !found {
		t.Fatalf("struct not found")
	}
	if f1Get.Name != f1.Name {
		t.Fatalf("Get expected to return user (%s), actual: (%s)", f1.Name, f1Get.Name)
	}

	f2Get := &fakeGet{}
	found = db.MustGet(f2Get, sqlq.Columns("name"), sqlq.Equal("user_id", f2.UserID))
	if !found {
		t.Fatalf("struct not found")
	}
	if f2Get.Name != f2.Name {
		t.Fatalf("Get expected to return user (%s), actual: (%s)", f2.Name, f2Get.Name)
	}
	if f2Get.UserID != "" {
		t.Fatalf("pgc fetched not only name column!")
	}

	f3Get := &fakeGet{}
	found = db.MustGet(f3Get, sqlq.Equal("user_id", "unknown"))
	if found {
		t.Fatalf("unexpected user found: (%s)", f3Get.Name)
	}
}

func TestMySQLColumnTypes(t *testing.T) {
	type columnTypes struct {
		ID        int
		Int8      int8 `sql_name:"int_8"`
		Int16     int16
		Int32     int32
		Int64     int64
		Uint      uint
		Uint8     uint8
		Uint16    uint16
		Uint32    uint32
		Uint64    uint64
		Float32   float32
		Float64   float64
		Bool      bool
		String    string
		CreatedAt time.Time
		Map       map[string]string
	}
	f1 := &columnTypes{
		Int8:      1,
		Int16:     2,
		Int32:     -3,
		Int64:     -4,
		Uint:      2,
		Uint8:     20,
		Uint32:    500,
		Uint64:    10000000,
		Float32:   -52.32,
		Float64:   23123123.2323,
		Bool:      true,
		String:    "string",
		CreatedAt: time.Now().Add(-24 * 365 * 10 * time.Hour),
		Map: map[string]string{
			"a": "b",
		},
	}
	db.MustCreateTable(f1)
	res := db.MustInsert(f1)

	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("fail get last id: %v", err)
	}
	t.Log(id)

	f1Get := &columnTypes{ID: int(id)}
	found := db.MustGet(f1Get)
	if !found {
		t.Fatalf("struct not found")
	}
	if f1Get.String != f1.String {
		t.Fatalf("string not the same (%s), actual: (%s)", f1.String, f1Get.String)
	}
	if f1Get.Int8 != f1.Int8 {
		t.Fatalf("Int8 not the same (%d), actual: (%d)", f1.Int8, f1Get.Int8)
	}
	if f1Get.Int16 != f1.Int16 {
		t.Fatalf("Int16 not the same (%d), actual: (%d)", f1.Int16, f1Get.Int16)
	}
	if f1Get.Int32 != f1.Int32 {
		t.Fatalf("Int32 not the same (%d), actual: (%d)", f1.Int32, f1Get.Int32)
	}
	if f1Get.Int64 != f1.Int64 {
		t.Fatalf("Int64 not the same (%d), actual: (%d)", f1.Int64, f1Get.Int64)
	}
	if f1Get.Uint != f1.Uint {
		t.Fatalf("Uint not the same (%d), actual: (%d)", f1.Uint, f1Get.Uint)
	}
	if f1Get.Uint8 != f1.Uint8 {
		t.Fatalf("Uint8 not the same (%d), actual: (%d)", f1.Uint8, f1Get.Uint8)
	}
	if f1Get.Uint16 != f1.Uint16 {
		t.Fatalf("Uint16 not the same (%d), actual: (%d)", f1.Uint16, f1Get.Uint16)
	}
	if f1Get.Uint32 != f1.Uint32 {
		t.Fatalf("Uint32 not the same (%d), actual: (%d)", f1.Uint32, f1Get.Uint32)
	}
	if f1Get.Uint64 != f1.Uint64 {
		t.Fatalf("Uint64 not the same (%d), actual: (%d)", f1.Uint64, f1Get.Uint64)
	}
	if f1Get.Float32 != f1.Float32 {
		t.Fatalf("Float32 not the same (%f), actual: (%f)", f1.Float32, f1Get.Float32)
	}
	if f1Get.Float64 != f1.Float64 {
		t.Fatalf("Float64 not the same (%f), actual: (%f)", f1.Float64, f1Get.Float64)
	}
	if f1Get.Bool != f1.Bool {
		t.Fatalf("Bool not the same (%v), actual: (%v)", f1.Bool, f1Get.Bool)
	}
	if f1Get.CreatedAt.Equal(f1.CreatedAt) {
		t.Fatalf("Time is not the same (%s), actual: (%s)", f1.CreatedAt, f1Get.CreatedAt)
	}
	if f1Get.Map == nil || f1Get.Map["a"] != f1.Map["a"] {
		t.Fatalf("Map is not the same (%v), actual: (%v)", f1.Map, f1Get.Map)
	}

	t.Run("nullable", func(t *testing.T) {
		type nullableColumnTypes struct {
			ID        int
			Int8      int8              `sql_name:"int_8" mw:"nullable"`
			Int16     int16             `mw:"nullable"`
			Int32     int32             `mw:"nullable"`
			Int64     int64             `mw:"nullable"`
			Uint      uint              `mw:"nullable"`
			Uint8     uint8             `mw:"nullable"`
			Uint16    uint16            `mw:"nullable"`
			Uint32    uint32            `mw:"nullable"`
			Uint64    uint64            `mw:"nullable"`
			Float32   float32           `mw:"nullable"`
			Float64   float64           `mw:"nullable"`
			Bool      bool              `mw:"nullable"`
			String    string            `mw:"nullable"`
			CreatedAt time.Time         `mw:"nullable"`
			Map       map[string]string `mw:"nullable"`
		}
		f1 := &nullableColumnTypes{
			Int8:      1,
			Int16:     2,
			Int32:     -3,
			Int64:     -4,
			Uint:      2,
			Uint8:     20,
			Uint32:    500,
			Uint64:    10000000,
			Float32:   -52.32,
			Float64:   23123123.2323,
			Bool:      true,
			String:    "string",
			CreatedAt: time.Now().Add(-24 * 365 * 10 * time.Hour),
			Map: map[string]string{
				"a": "b",
			},
		}
		db.MustCreateTable(f1)
		res := db.MustInsert(f1)

		id, err := res.LastInsertId()
		if err != nil {
			t.Fatalf("fail get last id: %v", err)
		}
		t.Log(id)

		f1Get := &nullableColumnTypes{ID: int(id)}
		found := db.MustGet(f1Get)
		if !found {
			t.Fatalf("struct not found")
		}
		if f1Get.String != f1.String {
			t.Fatalf("string not the same (%s), actual: (%s)", f1.String, f1Get.String)
		}
		if f1Get.Int8 != f1.Int8 {
			t.Fatalf("Int8 not the same (%d), actual: (%d)", f1.Int8, f1Get.Int8)
		}
		if f1Get.Int16 != f1.Int16 {
			t.Fatalf("Int16 not the same (%d), actual: (%d)", f1.Int16, f1Get.Int16)
		}
		if f1Get.Int32 != f1.Int32 {
			t.Fatalf("Int32 not the same (%d), actual: (%d)", f1.Int32, f1Get.Int32)
		}
		if f1Get.Int64 != f1.Int64 {
			t.Fatalf("Int64 not the same (%d), actual: (%d)", f1.Int64, f1Get.Int64)
		}
		if f1Get.Uint != f1.Uint {
			t.Fatalf("Uint not the same (%d), actual: (%d)", f1.Uint, f1Get.Uint)
		}
		if f1Get.Uint8 != f1.Uint8 {
			t.Fatalf("Uint8 not the same (%d), actual: (%d)", f1.Uint8, f1Get.Uint8)
		}
		if f1Get.Uint16 != f1.Uint16 {
			t.Fatalf("Uint16 not the same (%d), actual: (%d)", f1.Uint16, f1Get.Uint16)
		}
		if f1Get.Uint32 != f1.Uint32 {
			t.Fatalf("Uint32 not the same (%d), actual: (%d)", f1.Uint32, f1Get.Uint32)
		}
		if f1Get.Uint64 != f1.Uint64 {
			t.Fatalf("Uint64 not the same (%d), actual: (%d)", f1.Uint64, f1Get.Uint64)
		}
		if f1Get.Float32 != f1.Float32 {
			t.Fatalf("Float32 not the same (%f), actual: (%f)", f1.Float32, f1Get.Float32)
		}
		if f1Get.Float64 != f1.Float64 {
			t.Fatalf("Float64 not the same (%f), actual: (%f)", f1.Float64, f1Get.Float64)
		}
		if f1Get.Bool != f1.Bool {
			t.Fatalf("Bool not the same (%v), actual: (%v)", f1.Bool, f1Get.Bool)
		}
		if f1Get.CreatedAt.Equal(f1.CreatedAt) {
			t.Fatalf("Time is not the same (%s), actual: (%s)", f1.CreatedAt, f1Get.CreatedAt)
		}
		if f1Get.Map == nil || f1Get.Map["a"] != f1.Map["a"] {
			t.Fatalf("Map is not the same (%v), actual: (%v)", f1.Map, f1Get.Map)
		}

		insertQuery := `INSERT INTO nullable_column_types (` +
			"`int_8`,`int16`,`int32`,`int64`,`uint`,`uint8`,`uint16`,`uint32`,`uint64`,`float32`,`float64`,`bool`,`string`,`created_at`,`map`)" + `
		VALUES (NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL)`

		res, err = db.DB.Exec(insertQuery)
		if err != nil {
			t.Fatalf("fail execute query: %v", err)
		}
		id, err = res.LastInsertId()
		if err != nil {
			t.Fatalf("fail get inserted id: %v", err)
		}
		f2Get := &nullableColumnTypes{ID: int(id)}
		found = db.MustGet(f2Get)
		if !found {
			t.Fatalf("struct not found")
		}
		if f2Get.String != "" {
			t.Fatalf("string is not empty, actual (%s)", f2Get.String)
		}
		if f2Get.Int8 != 0 {
			t.Fatalf("Int8 is not empty, actual (%d)", f2Get.Int8)
		}
		if f2Get.Int16 != 0 {
			t.Fatalf("Int16 is not empty, actual (%d)", f2Get.Int16)
		}
		if f2Get.Int32 != 0 {
			t.Fatalf("Int32 is not empty, actual (%d)", f2Get.Int32)
		}
		if f2Get.Int64 != 0 {
			t.Fatalf("Int32 is not empty, actual (%d)", f2Get.Int64)
		}
		if f2Get.Uint != 0 {
			t.Fatalf("Uint is not empty, actual (%d)", f2Get.Uint)
		}
		if f2Get.Uint8 != 0 {
			t.Fatalf("Uint8 is not empty, actual (%d)", f2Get.Uint8)
		}
		if f2Get.Uint16 != 0 {
			t.Fatalf("Uint16 is not empty, actual (%d)", f2Get.Uint16)
		}
		if f2Get.Uint32 != 0 {
			t.Fatalf("Uint32 is not empty, actual (%d)", f2Get.Uint32)
		}
		if f2Get.Uint64 != 0 {
			t.Fatalf("Uint64 is not empty, actual (%d)", f2Get.Uint64)
		}
		if f2Get.Float32 != 0 {
			t.Fatalf("Float32 is not empty, actual (%f)", f2Get.Float32)
		}
		if f2Get.Float64 != 0 {
			t.Fatalf("Float64 is not empty, actual (%f)", f2Get.Float64)
		}
		if f2Get.Bool {
			t.Fatalf("Bool is not false")
		}
		if f2Get.CreatedAt.IsZero() {
			t.Fatalf("Time is not zero, actual (%s)", f2Get.CreatedAt)
		}
		if len(f2Get.Map) != 0 {
			t.Fatalf("Map is not empty (%v)", f2Get.Map)
		}

		var items []nullableColumnTypes
		if err := db.Select(&items); err != nil {
			t.Fatalf("fail get items: %v", err)
		}
	})
}

func TestMustInsert(t *testing.T) {
	type fakeInsert1 struct {
		ID   string
		Name string
	}
	f1 := &fakeInsert1{ID: RandomString(25), Name: "BoB"}
	db.MustCreateTable(f1)
	db.MustInsert(f1)
	f1Get := &fakeInsert1{ID: f1.ID}
	db.MustGet(f1Get)
	if f1Get.Name != f1.Name {
		t.Fatalf("TestInsertWPanic couldnt get back struct (%#v)", f1) // Dies
	}
	f2 := &fakeInsert1{ID: f1.ID, Name: "JiM"}

	func() { // Define a function then call it to fully encapsulate the error process
		defer func() {
			if err := recover(); err == nil {
				t.Errorf("TestInsertWPanic should have panic on duped struct (%#v)", f1)
			}
		}()
		// Should panic here
		db.MustInsert(f2)
	}()
}

func TestInsert(t *testing.T) {
	type fakeInsert2 struct {
		ID   string
		Name string
	}
	f1 := &fakeInsert2{ID: RandomString(25), Name: "BoBErr"}
	db.MustCreateTable(f1)
	_, err := db.Insert(f1)
	if err != nil {
		t.Fatalf("InsertErr's error on (%#v) should have been nil", f1)
	}
	f1Get := &fakeInsert2{ID: f1.ID}
	db.MustGet(f1Get)
	if f1Get.ID != f1.ID || f1Get.Name != f1.Name {
		t.Fatalf("TestInsertWErr couldnt get back struct (%#v)", f1) // Dies
	}
	f2 := &fakeInsert2{ID: f1.ID, Name: "JiMErr"}

	_, err = db.Insert(f2)
	if err == nil {
		t.Fatalf("InsertErr's error on (%#v) should NOT have been nil", f2)
	}

	// Let's also check the string code and message
	_, err = db.Insert(f2)
	// Code: (string) (len=5) "23505",
	// Message: (string) (len=66) "duplicate key value violates unique constraint \"fake_insert2_pkey\"",
	if !IsUniqueViolationError(err) {
		t.Errorf("TestInsertWErr InsertErrS expected code for unique violation, was (%s)", err)
	}

	t.Run("insert more that limit allows", func(t *testing.T) {
		items := make([]interface{}, 0, LimitInsert+1)
		for i := 0; i < LimitInsert+1; i++ {
			items = append(items, &fakeInsert2{ID: RandomString(5)})
		}

		if _, err := db.Insert(items...); err == nil {
			t.Fatalf("shouldn't allow insertion more than (%d) items", LimitInsert)
		}
	})
}

func TestUpdate(t *testing.T) {
	type fakeUpdate struct {
		ID        string
		CompanyID string
		Name      string
		Scores    int
		IsActive  bool
	}
	companyID1 := RandomString(20)
	companyID2 := RandomString(23)

	f1 := &fakeUpdate{ID: RandomString(25), CompanyID: companyID1, Name: "Bob", IsActive: true}
	f2 := &fakeUpdate{ID: RandomString(25), CompanyID: companyID2, Name: "John", IsActive: false}
	f3 := &fakeUpdate{ID: RandomString(25), CompanyID: companyID2, Name: "James", IsActive: true}
	db.MustCreateTable(f1)
	db.MustInsert(f1, f2, f3)

	t.Run("update by PK", func(t *testing.T) {
		f1.Scores = 200
		err := db.Update(f1)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	t.Run("update rows by column", func(t *testing.T) {
		num, err := db.UpdateRows(&fakeUpdate{}, Map{"scores": 50, "is_active": false}, sqlq.Equal("company_id", companyID2))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if num != 2 {
			t.Fatalf("2 items should be updated, actual num items: %d", num)
		}
		db.MustGet(f2)
		if f2.Scores != 50 {
			t.Errorf("f2 wasn't updated")
		}
		db.MustGet(f3)
		if f3.Scores != 50 {
			t.Errorf("f3 scores value wasn't updated")
		}
		if f3.IsActive {
			t.Errorf("f3 is_active value wasn't updated")
		}
		db.MustGet(f1)
		if f1.Scores == 50 {
			t.Errorf("f1 shouldn't be updated")
		}
	})
	t.Run("update rows no query", func(t *testing.T) {
		num, err := db.UpdateRows(&fakeUpdate{}, Map{"is_active": false})
		if err == nil {
			t.Fatalf("error expected if no options specified, rows affected: %d", num)
		}
	})
	t.Run("force update all rows", func(t *testing.T) {
		num, err := db.UpdateRows(&fakeUpdate{}, Map{"is_active": false}, sqlq.All())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if num != 1 {
			t.Fatalf("expected 1 row to be updated, actual rows num: (%d)", num)
		}
		db.MustGet(f1)
		if f1.IsActive {
			t.Errorf("f1 wasn't updated")
		}
	})
}

func TestDelete(t *testing.T) {
	type fakeDelete struct {
		ID        string
		CompanyID string
		Name      string
		Scores    int
		IsActive  bool
	}
	companyID1 := RandomString(20)
	companyID2 := RandomString(23)

	f1 := &fakeDelete{ID: RandomString(25), CompanyID: companyID1, Name: "Bob", IsActive: true}
	f2 := &fakeDelete{ID: RandomString(25), CompanyID: companyID2, Name: "John", IsActive: false}
	f3 := &fakeDelete{ID: RandomString(25), CompanyID: companyID2, Name: "James", IsActive: true}
	f4 := &fakeDelete{ID: RandomString(25), CompanyID: companyID2, Name: "James2", IsActive: true}
	f5 := &fakeDelete{ID: RandomString(25), CompanyID: RandomString(23), Name: "Forest", IsActive: false}
	db.MustCreateTable(f1)
	db.MustInsert(f1, f2, f3, f4, f5)

	t.Run("delete by PK", func(t *testing.T) {
		f1.Scores = 200
		err := db.Delete(f4)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	t.Run("delete rows", func(t *testing.T) {
		num, err := db.DeleteRows(&fakeDelete{}, sqlq.Equal("company_id", companyID2))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if num != 2 {
			t.Fatalf("2 items should be updated, actual num items: %d", num)
		}
		if found := db.MustGet(f2); found {
			t.Fatalf("f2 wasn't deleted")
		}
		if found := db.MustGet(f3); found {
			t.Fatalf("f3 wasn't deleted")
		}
	})
	t.Run("delete rows no query", func(t *testing.T) {
		num, err := db.DeleteRows(&fakeDelete{})
		if err == nil {
			t.Fatalf("error expected if no options specified, rows affected: %d", num)
		}
	})
	t.Run("force delete all rows", func(t *testing.T) {
		num, err := db.DeleteRows(&fakeDelete{}, sqlq.All())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if num != 2 {
			t.Fatalf("not all rows were updated, actual rows num: (%d)", num)
		}
		if found := db.MustGet(f1); found {
			t.Errorf("f1 wasn't deleted")
		}
	})
}

func TestCount(t *testing.T) {
	type countUser struct {
		ID   string
		Name string
	}

	db.MustCreateTable(&countUser{})
	if count := db.MustCount(&countUser{}); count != 0 {
		t.Fatalf("no users inserted, received count: %d", count)
	}

	u1 := &countUser{
		ID:   RandomString(30),
		Name: RandomString(10),
	}
	u2 := &countUser{
		ID:   RandomString(30),
		Name: RandomString(10),
	}

	db.MustInsert(u1, u2)

	if count := db.MustCount(&countUser{}); count != 2 {
		t.Fatalf("%d users should have been inserted, received count: %d", 2, count)
	}

	if count := db.MustCount(&countUser{}, sqlq.Equal("id", u1.ID)); count != 1 {
		t.Fatalf("%d user expected, received count: %d", 1, count)
	}
}

func TestSelectWithOpts(t *testing.T) {
	type fakeBlog struct {
		Name         string
		Descr        string
		ID           string
		PublishStart time.Time
	}

	blogs := []fakeBlog{
		{
			Name:         "blog1",
			Descr:        "descr1",
			ID:           "blog1",
			PublishStart: time.Now(),
		},
		{
			Name:         "blog2",
			Descr:        "descr2",
			ID:           "blog2",
			PublishStart: time.Now(),
		},
		{
			Name:         "blog3",
			Descr:        "descr3",
			ID:           "blog3",
			PublishStart: time.Now(),
		},
		{
			Name:         "blog4",
			Descr:        "descr3",
			ID:           "blog4",
			PublishStart: time.Now(),
		},
	}

	db.MustCreateTable(&fakeBlog{})
	for _, b := range blogs {
		db.MustInsert(&b)
	}

	t.Run("select by ID", func(t *testing.T) {
		var fetchedBlogs []fakeBlog
		db.Select(
			&fetchedBlogs,
			sqlq.Equal("id", blogs[0].ID),
		)
		if len(fetchedBlogs) == 0 {
			t.Fatal("no items found in select")
		}
		if len(fetchedBlogs) != 1 || fetchedBlogs[0].Name != blogs[0].Name {
			t.Errorf("select failed. Expected blog: (%v), actual: (%v)", blogs[0], fetchedBlogs[0])
		}
	})
	t.Run("select IN", func(t *testing.T) {
		var fetchedBlogs []fakeBlog
		db.Select(
			&fetchedBlogs,
			sqlq.IN("id", blogs[0].ID, blogs[1].ID, blogs[2].ID),
			sqlq.Limit(2),
		)
		if len(fetchedBlogs) != 2 {
			t.Fatalf("expected %d items, %d given", 2, len(fetchedBlogs))
		}
		if fetchedBlogs[0].Name != blogs[0].Name || fetchedBlogs[1].Name != blogs[1].Name {
			t.Errorf("select failed. Expected blogs: (%v), actual: (%v)", blogs[:2], fetchedBlogs)
		}
	})
	t.Run("limit offset", func(t *testing.T) {
		var fetchedBlogs []fakeBlog
		db.Select(
			&fetchedBlogs,
			sqlq.IN("id", blogs[0].ID, blogs[1].ID, blogs[2].ID),
			sqlq.Limit(2),
			sqlq.Offset(1),
		)
		if len(fetchedBlogs) != 2 {
			t.Fatalf("expected %d items, %d given", 2, len(fetchedBlogs))
		}
		if fetchedBlogs[0].Name != blogs[1].Name || fetchedBlogs[1].Name != blogs[2].Name {
			t.Errorf("select failed. Expected blogs: (%v), actual: (%v)", blogs[1:3], fetchedBlogs)
		}
	})
	t.Run("nested conditions", func(t *testing.T) {
		var fetchedBlogs []fakeBlog
		db.Select(
			&fetchedBlogs,
			sqlq.OR(
				sqlq.Equal("name", "blog4"),
				sqlq.AND(
					sqlq.Equal("descr", "descr3"),
					sqlq.IN("id", blogs[1].ID, blogs[2].ID),
				),
			),
		)
		if len(fetchedBlogs) != 2 {
			t.Fatalf("expected %d items, %d given", 2, len(fetchedBlogs))
		}
		if fetchedBlogs[0].Name != blogs[2].Name || fetchedBlogs[1].Name != blogs[3].Name {
			t.Errorf("select failed. Expected blogs: (%v), actual: (%v)", blogs[2:], fetchedBlogs)
		}
	})
	t.Run("custom orders", func(t *testing.T) {
		var fetchedBlogs []fakeBlog
		err := db.Select(
			&fetchedBlogs,
			sqlq.Order("descr", sqlq.DESC),
			sqlq.Order("name", sqlq.ASC),
			sqlq.Limit(3),
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(fetchedBlogs) != 3 {
			t.Fatalf("expected %d items, %d given", 3, len(fetchedBlogs))
		}
		if fetchedBlogs[0].ID != blogs[2].ID || fetchedBlogs[1].ID != blogs[3].ID || fetchedBlogs[2].ID != blogs[1].ID {
			t.Errorf("select failed. Expected blogs: (%v), actual: (%v)", []fakeBlog{blogs[2], blogs[3], blogs[1]}, fetchedBlogs)
		}
	})
	t.Run("like", func(t *testing.T) {
		b1 := &fakeBlog{
			Name:         "aa_name1",
			Descr:        "descr1",
			ID:           RandomString(30),
			PublishStart: time.Now(),
		}
		b2 := &fakeBlog{
			Name:         "aa_name2",
			Descr:        "descr1",
			ID:           RandomString(30),
			PublishStart: time.Now(),
		}
		b3 := &fakeBlog{
			Name:         "bb_name",
			Descr:        "descr1",
			ID:           RandomString(30),
			PublishStart: time.Now(),
		}
		db.MustInsert(b1, b2, b3)

		var fetchedBlogs []fakeBlog
		err := db.Select(
			&fetchedBlogs,
			sqlq.Like("name", "aa_%"),
			sqlq.Order("name", sqlq.ASC),
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(fetchedBlogs) != 2 {
			t.Fatalf("expected %d items, %d given", 2, len(fetchedBlogs))
		}
		if fetchedBlogs[0].ID != b1.ID || fetchedBlogs[1].ID != b2.ID {
			t.Errorf("select failed. Expected blogs: (%v), actual: (%v)", []fakeBlog{*b1, *b2}, fetchedBlogs)
		}
	})
	t.Run("raw query", func(t *testing.T) {
		t.Run("raw select", func(t *testing.T) {
			var fetchedBlogs []fakeBlog
			err := db.Select(
				&fetchedBlogs,
				sqlq.OR(
					sqlq.IN("id", blogs[0].ID, blogs[1].ID),
					sqlq.Raw("name = ?", "blog4"),
				),
			)
			if err != nil {
				t.Fatalf("fail get blogs: %v", err)
			}
			if len(fetchedBlogs) != 3 {
				t.Fatalf("expected %d items, %d given", 3, len(fetchedBlogs))
			}
			if fetchedBlogs[0].Name != blogs[0].Name || fetchedBlogs[1].Name != blogs[1].Name {
				t.Errorf("select failed. Expected blogs: (%v), actual: (%v)", blogs[:2], fetchedBlogs)
			}
		})
		t.Run("only raw query", func(t *testing.T) {
			type JobTest struct {
				ID           string
				Name         string
				Frequency    int
				LastExecuted time.Time
			}
			j1 := &JobTest{
				ID:           RandomString(10),
				Name:         "name-111",
				Frequency:    1,
				LastExecuted: time.Now().Add(-2 * time.Minute).UTC(),
			}
			j2 := &JobTest{
				ID:           RandomString(10),
				Name:         "name-222",
				Frequency:    3,
				LastExecuted: time.Now().Add(-1 * time.Minute).UTC(),
			}
			j3 := &JobTest{
				ID:           RandomString(10),
				Name:         "name-333",
				Frequency:    2,
				LastExecuted: time.Now().UTC(),
			}
			j4 := &JobTest{
				ID:           RandomString(10),
				Name:         "name-444",
				Frequency:    1,
				LastExecuted: time.Now().Add(-5 * time.Minute).UTC(),
			}
			db.MustCreateTable(&JobTest{})

			_, err := db.Insert(j1, j2, j3, j4)
			if err != nil {
				t.Fatalf("fail insert jobs: %v", err)
			}

			rawQuery := "last_executed <= NOW() - INTERVAL ? SECOND"
			var items []JobTest
			err = db.Select(
				&items,
				sqlq.Raw(rawQuery, 90),
				sqlq.Order("name", sqlq.ASC),
			)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(items) != 2 {
				t.Fatalf("expected items num: %d, got %d", 2, len(items))
			}
			if items[0].ID != j1.ID {
				t.Fatalf("expected item #1 ID: %s, actual item: %v", j1.ID, items[0])
			}
			if items[1].ID != j4.ID {
				t.Fatalf("expected item #2 ID: %s, actual item: %v", j4.ID, items[1])
			}
		})
	})
	t.Run("ensure pointer data is correct", func(t *testing.T) {
		type nestedData struct {
			Name, Title string
		}

		type pointerData struct {
			ID     string
			Name   string
			Map    map[string]string
			Slice  []int
			Nested nestedData
		}
		db.MustCreateTable(&pointerData{})

		i1 := &pointerData{
			ID:     "111",
			Name:   "name 1",
			Map:    map[string]string{"1": "1"},
			Slice:  []int{1, 2, 3},
			Nested: nestedData{"11", "11"},
		}
		i2 := &pointerData{
			ID:     "222",
			Name:   "name 2",
			Map:    map[string]string{"1": "2"},
			Slice:  []int{4, 5, 6},
			Nested: nestedData{"22", "22"},
		}
		db.MustInsert(i1, i2)

		var res []pointerData
		err := db.Select(&res)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(res) != 2 {
			t.Fatalf("expected %d items, %d given", 2, len(res))
		}
		res1, res2 := res[0], res[1]
		if res1.ID != i1.ID || res1.Name != i1.Name || res1.Map == nil || res1.Map["1"] != i1.Map["1"] || len(res1.Slice) == 0 || res1.Slice[0] != i1.Slice[0] {
			t.Errorf("result data is not the same as inserted: expected (%v), actual (%v)", i1, res1)
		}
		if res1.Nested.Name != i1.Nested.Name {
			t.Errorf("result nested data is not the same as inserted: expected (%v), actual (%v)", i1, res1)
		}
		if res2.ID != i2.ID || res2.Name != i2.Name || res2.Map == nil || res2.Map["1"] != i2.Map["1"] || len(res2.Slice) == 0 || res2.Slice[0] != i2.Slice[0] {
			t.Errorf("result data is not the same as inserted: expected (%v), actual (%v)", i2, res2)
		}
		if res2.Nested.Name != i2.Nested.Name {
			t.Errorf("result nested data is not the same as inserted: expected (%v), actual (%v)", i2, res2)
		}
	})
}

func TestJoin(t *testing.T) {
	type subscriptionJoin struct {
		ID     string
		UserID string
		URL    string
	}
	type orderJoin struct {
		ID     string
		UserID string
		Total  int
	}
	type socialJoin struct {
		UserID    string `mw:"pk"`
		FBID      string `sql_name:"fb_id"`
		TwitterID string
	}

	type userJoin struct {
		ID            string
		Name          string
		Subscriptions []subscriptionJoin `mw:"join"`
		Orders        []orderJoin        `mw:"join"`
		Social        *socialJoin        `mw:"join"`
		PublishStart  time.Time
	}

	user1 := userJoin{ID: "u111", Name: "user1", PublishStart: time.Now()}
	user2 := userJoin{ID: "u222", Name: "user2", PublishStart: time.Now()}
	user3 := userJoin{ID: "u333", Name: "user3", PublishStart: time.Now()}

	sub1 := subscriptionJoin{ID: "s111", UserID: user1.ID, URL: "https://whowhatwhear.com/url1"}
	sub2 := subscriptionJoin{ID: "s222", UserID: user1.ID, URL: "https://whowhatwhear.com/article"}
	sub3 := subscriptionJoin{ID: "s333", UserID: user1.ID, URL: "url2"}
	sub4 := subscriptionJoin{ID: "s444", UserID: user3.ID, URL: "https://whowhatwhear.com/another-article"}

	o1 := &orderJoin{ID: "o111", UserID: user1.ID, Total: 111}
	o2 := &orderJoin{ID: "o222", UserID: user1.ID, Total: 500}
	o3 := &orderJoin{ID: "o333", UserID: user3.ID, Total: 200}
	o4 := &orderJoin{ID: "o444", UserID: user2.ID, Total: 700}

	s1 := &socialJoin{UserID: user1.ID, FBID: "1-1", TwitterID: "2-2"}
	s2 := &socialJoin{UserID: user2.ID, FBID: "2-2", TwitterID: "3-3"}

	userRows := []userJoin{user1, user2, user3}
	subRows := []subscriptionJoin{sub1, sub2, sub3, sub4}

	db.MustCreateTable(&userJoin{})
	db.MustCreateTable(&subscriptionJoin{})
	db.MustCreateTable(&orderJoin{})
	db.MustCreateTable(&socialJoin{})
	for _, r := range userRows {
		db.MustInsert(&r)
	}
	for _, r := range subRows {
		db.MustInsert(&r)
	}
	db.MustInsert(o1, o2, o3, o4)
	db.MustInsert(s1, s2)

	const userJoinSubscription = "user_join.id = subscription_join.user_id"
	const userJoinOrder = "user_join.id = order_join.user_id"
	const userJoinSocial = "user_join.id = social_join.user_id"

	t.Run("one join", func(t *testing.T) {
		var fetchedRows []userJoin
		err := db.Select(
			&fetchedRows,
			sqlq.Columns("id", "name"),
			sqlq.Join(&subscriptionJoin{}, userJoinSubscription, "url"),
			sqlq.Order("name", sqlq.ASC),
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(fetchedRows) != 3 {
			t.Fatalf("expected %d items, %d given", 3, len(fetchedRows))
		}
		if len(fetchedRows[0].Subscriptions) != 3 {
			t.Fatalf("row 1 expected to have 3 subscriptions: actual: (%v)", fetchedRows[0].Subscriptions)
		}
		if fetchedRows[0].Subscriptions[0].ID != "s111" {
			t.Errorf("1st subscription of 1st user expected to be (%v), actual: (%v)", sub1, fetchedRows[0].Subscriptions[0])
		}
		if len(fetchedRows[1].Subscriptions) != 0 {
			t.Errorf("row 1 expected to have 0 subscription: actual: (%v)", fetchedRows[1])
		}
		if len(fetchedRows[2].Subscriptions) != 1 {
			t.Errorf("row 1 expected to have 1 subscription: actual: (%v)", fetchedRows[2])
		}
	})
	t.Run("multiple join", func(t *testing.T) {
		var fetchedRows []userJoin
		err := db.Select(
			&fetchedRows,
			sqlq.Columns("id", "name"),
			sqlq.Join(&subscriptionJoin{}, userJoinSubscription, "url"),
			sqlq.Join(&orderJoin{}, userJoinOrder, "total"),
			sqlq.Like("url", "https://whowhatwhear.com/%"),
			sqlq.GreaterOrEqual("total", 500),
			sqlq.Order("name", sqlq.ASC),
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(fetchedRows) != 1 {
			t.Fatalf("expected %d items, %d given", 1, len(fetchedRows))
		}
		if len(fetchedRows[0].Subscriptions) != 2 {
			t.Fatalf("row 1 expected to have 2 subscriptions: actual: (%v)", fetchedRows[0].Subscriptions)
		}
		if len(fetchedRows[0].Orders) != 1 {
			t.Fatalf("row 1 expected to have 1 orders: actual: (%v)", fetchedRows[0].Orders)
		}
	})
	t.Run("one to one join", func(t *testing.T) {
		var fetchedRows []userJoin
		err := db.Select(
			&fetchedRows,
			sqlq.Columns("id", "name"),
			sqlq.Join(&socialJoin{}, userJoinSocial),
			sqlq.Join(&orderJoin{}, userJoinOrder, "total"),
			sqlq.GreaterOrEqual("total", 200),
			sqlq.Order("name", sqlq.ASC),
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(fetchedRows) != 3 {
			t.Fatalf("expected %d items, %d given", 2, len(fetchedRows))
		}
		if len(fetchedRows[0].Orders) != 1 {
			t.Fatalf("row 1 expected to have 1 orders: actual: (%v)", fetchedRows[0].Orders)
		}
		if len(fetchedRows[1].Orders) != 1 {
			t.Fatalf("row 1 expected to have 1 orders: actual: (%v)", fetchedRows[1].Orders)
		}
		if fetchedRows[0].Social.TwitterID != s1.TwitterID {
			t.Errorf("row 1 expected twitter ID %s: actual: (%s)", s1.TwitterID, fetchedRows[0].Social.TwitterID)
		}
		if fetchedRows[0].Social.FBID != s1.FBID {
			t.Errorf("row 1 expected fb ID %s: actual: (%s)", s1.FBID, fetchedRows[0].Social.FBID)
		}
		if fetchedRows[1].Social.TwitterID != s2.TwitterID {
			t.Errorf("row 2 expected twitter ID %s: actual: (%s)", s2.TwitterID, fetchedRows[1].Social.TwitterID)
		}
		if fetchedRows[1].Social.FBID != s2.FBID {
			t.Errorf("row 2 expected fb ID %s: actual: (%s)", s2.FBID, fetchedRows[1].Social.FBID)
		}
		if fetchedRows[2].Social != nil {
			t.Errorf("row 3 expected to have empty social, actual: (%v)", fetchedRows[2].Social)
		}
	})

	t.Run("many to many join", func(t *testing.T) {
		type lessonJoin struct {
			ID   string
			Name string
		}

		type studentJoin struct {
			ID      string
			Name    string
			Lessons []lessonJoin `mw:"join"`
		}

		type studentLessonJoin struct {
			ID        string
			StudentID string
			LessonID  string
			MW        struct{} `mw:"many_to_many"`
		}

		db.MustCreateTable(&lessonJoin{})
		db.MustCreateTable(&studentJoin{})
		db.MustCreateTable(&studentLessonJoin{})

		l1 := &lessonJoin{ID: "111", Name: "lesson1"}
		l2 := &lessonJoin{ID: "222", Name: "lesson2"}
		l3 := &lessonJoin{ID: "333", Name: "lesson3"}
		l4 := &lessonJoin{ID: "444", Name: "lesson4"}
		db.MustInsert(l1, l2, l3, l4)

		s1 := &studentJoin{ID: "111", Name: "student1"}
		s2 := &studentJoin{ID: "222", Name: "student2"}
		s3 := &studentJoin{ID: "333", Name: "student3"}
		s4 := &studentJoin{ID: "444", Name: "student4"}
		db.MustInsert(s1, s2, s3, s4)

		sl1 := &studentLessonJoin{ID: "1", StudentID: "111", LessonID: "111"}
		sl2 := &studentLessonJoin{ID: "2", StudentID: "111", LessonID: "222"}
		sl3 := &studentLessonJoin{ID: "3", StudentID: "222", LessonID: "333"}
		sl4 := &studentLessonJoin{ID: "4", StudentID: "333", LessonID: "444"}
		db.MustInsert(sl1, sl2, sl3, sl4)

		const studentJoinStudentLesson = "student_join.id = student_lesson_join.student_id"
		const lessonJoinStudentLesson = "lesson_join.id = student_lesson_join.lesson_id"

		var fetchedRows []studentJoin
		err := db.Select(
			&fetchedRows,
			sqlq.Join(&studentLessonJoin{}, studentJoinStudentLesson),
			sqlq.Join(&lessonJoin{}, lessonJoinStudentLesson),
			sqlq.NotEqual(GetColumnName(&lessonJoin{}, "name"), "lesson1"),
			sqlq.Order(GetColumnName(&lessonJoin{}, "name"), sqlq.ASC),
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(fetchedRows) != 3 {
			t.Fatalf("expected %d items, %d given", 3, len(fetchedRows))
		}
		if len(fetchedRows[0].Lessons) != 1 {
			t.Fatalf("row 1 expected to have 2 lessons: actual: (%v)", fetchedRows[0].Lessons)
		}
		if fetchedRows[0].Lessons[0].Name != l2.Name {
			t.Fatalf("row 1 lesson expected to be (%s), actual: (%s)", fetchedRows[0].Lessons[0].Name, l2.Name)
		}
		if len(fetchedRows[1].Lessons) != 1 {
			t.Fatalf("row 2 expected to have 1 lessons: actual: (%v)", fetchedRows[1].Lessons)
		}
		if fetchedRows[1].Lessons[0].Name != l3.Name {
			t.Fatalf("row 2 lesson expected to be (%s), actual: (%s)", fetchedRows[1].Lessons[0].Name, l3.Name)
		}
		if len(fetchedRows[2].Lessons) != 1 {
			t.Fatalf("row 3 expected to have 1 lessons: actual: (%v)", fetchedRows[2].Lessons)
		}
		if fetchedRows[2].Lessons[0].Name != l4.Name {
			t.Fatalf("row 3 lesson expected to be (%s), actual: (%s)", fetchedRows[2].Lessons[0].Name, l4.Name)
		}
	})
	t.Run("join on get", func(t *testing.T) {
		var u userJoin
		found, err := db.Get(
			&u,
			sqlq.Equal(GetColumnName(&userJoin{}, "id"), user1.ID),
			sqlq.Join(&socialJoin{}, userJoinSocial),
			sqlq.Join(&orderJoin{}, userJoinOrder, "total"),
			sqlq.GreaterOrEqual("total", 200),
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !found {
			t.Fatalf("user %s not found", u.ID)
		}
		if len(u.Orders) != 1 {
			t.Fatalf("row 1 expected to have 1 orders: actual: (%v)", u.Orders)
		}
		if u.Social.TwitterID != s1.TwitterID {
			t.Errorf("row 1 expected twitter ID %s: actual: (%s)", s1.TwitterID, u.Social.TwitterID)
		}
		if u.Social.FBID != s1.FBID {
			t.Errorf("row 1 expected fb ID %s: actual: (%s)", s1.FBID, u.Social.FBID)
		}

		var notFoundUser userJoin
		found, err = db.Get(
			&notFoundUser,
			sqlq.Equal(GetColumnName(&userJoin{}, "id"), util.NewGuid()),
			sqlq.Join(&socialJoin{}, userJoinSocial),
			sqlq.Join(&orderJoin{}, userJoinOrder, "total"),
			sqlq.GreaterOrEqual("total", 200),
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if found {
			t.Errorf("expect to not found user, actual user: %v", notFoundUser)
		}
	})
}

var ranStrSetAlphaNum = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

func init() {
	// Seed the random num gen, once for app
	rand.Seed(time.Now().UTC().UnixNano())
}

func RandomInt(min, max int) int {
	/* We will give back min - max inclusive.
	   0,0 always gives back 0
	   1,1 always gives back 1
	   1,2 gives back either 1 or 2
	   etc
	   1,0 panics (Intn panics on negative #)
	*/
	return min + rand.Intn(max+1-min)
}

func RandomString(length uint) string {
	if length < 1 {
		msg := "Cant ask for random string of length less than 1"
		panic(msg)
	}
	bytes := make([]byte, length)
	for i := range bytes {
		bytes[i] = ranStrSetAlphaNum[rand.Intn(len(ranStrSetAlphaNum))]
	}
	return string(bytes)
}
