package mwear

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
)

func TestPathDirExists(t *testing.T) {
	tmpDir := os.TempDir()
	if !PathExists(tmpDir) {
		t.Errorf("TmpDir (%s) should have existed", tmpDir)
	}
	if PathExists("/hoogaboogabuhbollaoughskiiiii/") {
		t.Error("Dir hoogaboogabuhbollaoughskiiiii should NOT have existed")
	}
}

func TestPathFileExists(t *testing.T) {
	tmpDir := os.TempDir()

	newFilePath := fmt.Sprintf("%s/golangtestfile-%d", tmpDir, RandomInt(10000000, 999999999))
	fakeFile := tmpDir + "/lkajsldkfjsdlkfjalskdfj.out"

	err := ioutil.WriteFile(newFilePath, []byte("Yarlow"), 0777)
	if err != nil {
		t.Fatalf("Failed to create a tmp file (%s), err: (%s)", newFilePath, err)
	}

	if !PathExists(tmpDir) {
		t.Errorf("Tmp file (%s) should have existed", newFilePath)
	}
	if PathExists(fakeFile) {
		t.Errorf("File (%s) should NOT have existed", fakeFile)
	}
}

func TestMakeOrderBy(t *testing.T) {
	type OrderModel struct {
		ID           string
		CustomeField string `sql_name:"field_custom"`
		UpdatedAt    string
	}

	tests := []struct {
		name                     string
		sortBy                   string
		wantField, wantDirection string
		wantOk                   bool
	}{
		{
			name:   "empty name",
			sortBy: "",
		},
		{
			name:   "invalid field name",
			sortBy: "invalid name",
		},
		{
			name:          "default direction",
			sortBy:        "id",
			wantField:     "id",
			wantDirection: "ASC",
			wantOk:        true,
		},
		{
			name:          "with spaces",
			sortBy:        " id ",
			wantField:     "id",
			wantDirection: "ASC",
			wantOk:        true,
		},
		{
			name:          "camelcased",
			sortBy:        "UpdatedAt",
			wantField:     "updated_at",
			wantDirection: "ASC",
			wantOk:        true,
		},
		{
			name:          "custom name",
			sortBy:        "-CustomeField",
			wantField:     "field_custom",
			wantDirection: "DESC",
			wantOk:        true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotField, gotDirection, gotOk := MakeOrderBy(&OrderModel{}, tt.sortBy)
			if gotField != tt.wantField {
				t.Errorf("MakeOrderBy() gotField = %v, want %v", gotField, tt.wantField)
			}
			if gotDirection != tt.wantDirection {
				t.Errorf("MakeOrderBy() gotDirection = %v, want %v", gotField, tt.wantDirection)
			}
			if gotOk != tt.wantOk {
				t.Errorf("MakeOrderBy() gotOk = %v, want %v", gotOk, tt.wantOk)
			}
		})
	}
}
