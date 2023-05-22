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

reflect 的一些方法

```go
reflect.ValueOf() 获取指针对应的反射值。
reflect.Indirect() 获取指针指向的对象的反射值。
(reflect.Type).Name() 返回类名(字符串)。
(reflect.Type).Field(i) 获取第 i 个成员变量。
```
