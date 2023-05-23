package session

import (
	"database/sql"
	"geeorm/dialect"

	_ "github.com/mattn/go-sqlite3"
)

type User struct {
	Name string `geeorm:"PRIMARY KEY"`
	Age  int
}

func NewSession() *Session {
	db, _ := sql.Open("sqlite3", "../db/test.db")
	dia, _ := dialect.GetDialect("sqlite3")
	return New(db, dia)
}