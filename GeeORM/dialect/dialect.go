package dialect

import "reflect"

/* --------------------------- Go 语言的类型映射为数据库中的类型 --------------------------- */

/**
 * 抽象出各个数据库差异的部分
 */

var dialectMap = map[string]Dialect{}

type Dialect interface {
	// Go 语言的类型转换为该数据库的数据类型
	DataTypeOf(reflect.Value) string
	// 返回某个表是否存在的 SQL 语句
	TableExistSQL(tableName string) (string, []interface{})
}

// 注册 dialect 实例
func RegisterDialect(name string, dialect Dialect) {
	dialectMap[name] = dialect
}

// 获取 dialect 实例
func GetDialect(name string) (dialect Dialect, ok bool) {
	dialect, ok = dialectMap[name]
	return
}
