package main

import (
	"fmt"
	"geeorm/geeorm"

	_ "github.com/mattn/go-sqlite3"
)

func test1() {
	engine, _ := geeorm.NewEngine("sqlite3", "./gee.db")
	defer engine.Close()
	session := engine.NewSession()

	_, _ = session.Raw("DROP TABLE IF EXISTS User;").Exec()
    _, _ = session.Raw("CREATE TABLE User(Name text);").Exec()
    _, _ = session.Raw("CREATE TABLE User(Name text);").Exec()
	result, _ := session.Raw("INSERT INTO User(`Name`) values (?), (?)", "Tom", "Sam").Exec()
	count, _ := result.RowsAffected()
	fmt.Printf("Exec success, %d affected\n", count)
}

func main() {
	test1()
}