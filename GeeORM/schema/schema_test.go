package schema

import (
	"geeorm/dialect"
	"testing"
)

type User struct {
	Name string `geeorm:"PRIMARY KEY"`
	Age int
}

func TestParse(t *testing.T) {
	s, _:=dialect.GetDialect("sqlite3")
	schema := Parse(&User{}, s)

	if schema.Name != "User" || len(schema.Fields) != 2 {
		t.Fatal("failed to parse User struct")
	}
	if schema.GetField("Name").Tag != "PRIMARY KEY" {
		t.Fatal("failed to parse primary key")
	}
}