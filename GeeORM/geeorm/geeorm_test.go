package geeorm

import (
	"errors"
	"geeorm/session"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

type User struct {
	Name string `geeorm:"PRIMARY KEY"`
	Age  int
}

func OpenDB(t *testing.T) *Engine {
	t.Helper()
	engine, err := NewEngine("sqlite3", "../db/test.db")
	if err != nil {
		t.Fatal(err)
	}
	return engine
}

func TestEngine_Transaction(t *testing.T) {
	t.Run("rollback", func(t *testing.T) {
		engine := OpenDB(t)
		defer engine.Close()
		s := engine.NewSession()
		_ = s.Model(&User{}).DropTable()

		_, err := engine.Transaction(func(s *session.Session) (result interface{}, err error) {
			_ = s.Model(&User{}).CreateTable()
			_, err = s.Insert(&User{"Tom", 18})
			return nil, errors.New("Error")
		})

		if err == nil || s.HasTable() {
			t.Fatal("failed to rollback")
		}
	})

	t.Run("commit", func(t *testing.T) {
		engine := OpenDB(t)
		defer engine.Close()
		s := engine.NewSession()
		_ = s.Model(&User{}).DropTable()

		_, err := engine.Transaction(func(s *session.Session) (result interface{}, err error) {
			_ = s.Model(&User{}).CreateTable()
			_, err = s.Insert(&User{"Tom", 18})
			return
		})

		var user User
		_ = s.First(&user)

		if err != nil || user.Name != "Tom" {
			t.Fatal("failed to rollback")
		}
	})
}
