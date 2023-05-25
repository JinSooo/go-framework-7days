# GeeORM

> start: 2023-5-22 10:15PM

GeeORM 的设计主要参考了 xorm，一些细节上的实现参考了 gorm。GeeORM 的目的主要是了解 ORM 框架设计的原理，具体实现上鲁棒性做得不够，一些复杂的特性，例如 gorm 的关联关系，xorm 的读写分离没有实现。目前支持的特性有：

- 表的创建、删除、迁移。
- 记录的增删查改，查询条件的链式操作。
- 单一主键的设置(primary key)。
- 钩子(在创建/更新/删除/查找之前或之后)
- 事务(transaction)。
- ...

## SQLite

SQLite 是一款轻量级的，遵守 ACID 事务原则的关系型数据库。SQLite 可以直接嵌入到代码中，不需要像 MySQL、PostgreSQL 需要启动独立的服务才能使用。SQLite 将数据存储在单一的磁盘文件中，使用起来非常方便。也非常适合初学者用来学习关系型数据的使用。GeeORM 的所有的开发和测试均基于 SQLite。

### SQLite 在 Windows 的安装使用

> SQLite 下载 (https://www.sqlite.org/download.html)
>
> 下载 sqlite-dll-win64-x64-3420000.zip、sqlite-tools-win32-x86-3420000.zip

解压、安装、配置环境变量

### go 中配置 SQLite

> 注意：在 Windows 中，SQLite 需要依赖 gcc，所以还需要 mingw

配置 CGO

```bash
go env -w CGO_ENABLED=1
```

测试代码

```go
package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	db, _ := sql.Open("sqlite3", "./db/gee.db")
	defer func() { _ = db.Close() }()
	_, _ = db.Exec("CREATE TABLE User(Name text);")
	result, err := db.Exec("INSERT INTO User(`Name`) values (?), (?)", "Tom", "Sam")
	if err == nil {
		affected, _ := result.RowsAffected()
		log.Println(affected)
	}
	row := db.QueryRow("SELECT Name FROM User LIMIT 1")
	var name string
	if err := row.Scan(&name); err == nil {
		log.Println(name)
	}
}
```

结果

```
2023/05/22 22:52:16 2
2023/05/22 22:52:16 Tom
```

## 如何根据任意类型的指针，得到其对应的结构体的信息。

这涉及到了 Go 语言的反射机制(reflect)，通过反射，可以获取到对象对应的结构体名称，成员变量、方法等信息，例如：

```go
type Account struct {
    Username string
    Password string
}

typ := reflect.Indirect(reflect.ValueOf(&Account{})).Type()
fmt.Println(typ.Name()) // Account

for i := 0; i < typ.NumField(); i++ {
    field := typ.Field(i)
    fmt.Println(field.Name) // Username Password
}
```

### reflect 的一些方法

```go
reflect.ValueOf() 获取指针对应的反射值。
reflect.Indirect() 获取指针指向的对象的反射值。
(reflect.Type).Name() 返回类名(字符串)。
(reflect.Type).Field(i) 获取第 i 个成员变量。
```

## 特性 👇👇👇

## 基本使用

```go
var (
	user1 = &User{"Tom", 18}
	user2 = &User{"Sam", 25}
	user3 = &User{"Jack", 25}
)

func testRecordInit(t *testing.T) *Session {
	t.Helper()

	session := NewSession().Model(&User{})
	err1 := session.DropTable()
	err2 := session.CreateTable()
	_, err3 := session.Insert(user1, user2)

	if err1 != nil || err2 != nil || err3 != nil {
		t.Fatal("failed init test records")
	}

	return session
}

func TestInsert(t *testing.T) {
	session := testRecordInit(t)

	affected, err := session.Insert(user3)
	if err != nil || affected != 1 {
		t.Fatal("failed to create record")
	}
}

func TestFind(t *testing.T) {
	session := testRecordInit(t)

	var users []User
	err := session.Find(&users)
	if err != nil || len(users) != 2 {
		t.Fatal("failed to query all")
	}
}

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

func TestSession_Limit(t *testing.T) {
	s := testRecordInit(t)
	var users []User
	err := s.Limit(1).Find(&users)
	if err != nil || len(users) != 1 {
		t.Fatal("failed to query with limit condition")
	}
}

func TestSession_Update(t *testing.T) {
	s := testRecordInit(t)
	affected, _ := s.Where("Name = ?", "Tom").Update("Age", 30)
	u := &User{}
	_ = s.OrderBy("Age DESC").First(u)

	if affected != 1 || u.Age != 30 {
		t.Fatal("failed to update")
	}
}

func TestSession_DeleteAndCount(t *testing.T) {
	s := testRecordInit(t)
	affected, _ := s.Where("Name = ?", "Tom").Delete()
	count, _ := s.Count()

	if affected != 1 || count != 1 {
		t.Fatal("failed to delete or count")
	}
}
```

## Hooks

```go
type Account struct {
	ID       int `geeorm:"PRIMARY KEY"`
	Password string
}

func (account *Account) BeforeInsert(s *Session) error {
	log.Info("before inert", account)
	account.ID += 1000
	return nil
}

func (account *Account) AfterQuery(s *Session) error {
	log.Info("after query", account)
	account.Password = "******"
	return nil
}

func TestSession_CallMethod(t *testing.T) {
	s := NewSession().Model(&Account{})
	_ = s.DropTable()
	_ = s.CreateTable()
	_, _ = s.Insert(&Account{1, "123456"}, &Account{2, "qwerty"})

	u := &Account{}

	err := s.First(u)
	if err != nil || u.ID != 1001 || u.Password != "******" {
		t.Fatal("Failed to call hooks after query, got", u)
	}
}
```

## 事务

```go
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
```

## 数据库迁移

```go
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

func TestEngine_Migrate(t *testing.T) {
	engine := OpenDB(t)
	defer engine.Close()
	s := engine.NewSession()
	_, _ = s.Raw("DROP TABLE IF EXISTS User;").Exec()
	_, _ = s.Raw("CREATE TABLE User(Name text PRIMARY KEY, XXX integer);").Exec()
	_, _ = s.Raw("INSERT INTO User(`Name`) values (?), (?)", "Tom", "Sam").Exec()
	engine.Migrate(&User{})

	rows, _ := s.Raw("SELECT * FROM User").QueryRows()
	columns, _ := rows.Columns()
	if !reflect.DeepEqual(columns, []string{"Name", "Age"}) {
		t.Fatal("Failed to migrate table User, got columns", columns)
	}
}
```
