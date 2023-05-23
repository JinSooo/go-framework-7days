package session

import (
	"fmt"
	"geeorm/log"
	"geeorm/schema"
	"reflect"
	"strings"
)

/* ------------------------------- 操作数据库表相关的代码 ------------------------------ */

// 更新refTable
func (session *Session) Model(value interface{}) *Session {
	// 当结构体类型改变时，更新refTable，重新解析出Schema
	if session.refTable == nil || reflect.TypeOf(value) != reflect.TypeOf(session.refTable.Model) {
		session.refTable = schema.Parse(value, session.dialect)
	}

	return session
}

func (session *Session) RefTable() *schema.Schema {
	if session.refTable == nil {
		log.Error("Model is not set")
	}
	return session.refTable
}

// 建表
func (session *Session) CreateTable() error {
	table := session.RefTable()
	var columns []string

	// 格式化获取所有列
	for _, field := range table.Fields {
		columns = append(columns, fmt.Sprintf("%s %s %s", field.Name, field.Type, field.Tag))
	}

	desc := strings.Join(columns, ",")
	_, err := session.Raw(fmt.Sprintf("CREATE TABLE %s (%s)", table.Name, desc)).Exec()

	return err
}

// 删表
func (session *Session) DropTable() error {
	_, err := session.Raw(fmt.Sprintf("DROP TABLE IF EXISTS %s", session.RefTable().Name)).Exec()

	return err
}

// 表是否存在
func (session *Session) HasTable() bool {
	sql, values := session.dialect.TableExistSQL(session.RefTable().Name)
	row := session.Raw(sql, values...).QueryRow()

	var tmp string
	// 获取name
	_ = row.Scan(&tmp)

	return tmp == session.refTable.Name
}
