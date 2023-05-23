package session

import (
	"fmt"
	"testing"
)


func TestTable(t *testing.T) {
	gee := NewSession()
	fmt.Printf("gee: %v\n", gee)
	s := gee.Model(&User{})
	_ = s.DropTable()
	_ = s.CreateTable()
	if !s.HasTable() {
		t.Fatal("Failed to create table User")
	}
}