package clause

import "strings"

/* ------------------------- 实现结构体 Clause 拼接各个独立的子句 ------------------------- */

type Type int

const (
	INSERT Type = iota
	VALUES
	SELECT
	LIMIT
	WHERE
	ORDERBY
)

type Clause struct {
	sql     map[Type]string
	sqlVars map[Type][]interface{}
}

// 根据 Type 调用对应的 generator，生成该子句对应的 SQL 语句
func (clause *Clause) Set(name Type, vars ...interface{}) {
	if clause.sql == nil {
		clause.sql = make(map[Type]string)
		clause.sqlVars = make(map[Type][]interface{})
	}

	// 调用子句生成器
	sql, vars := generators[name](vars...)
	clause.sql[name] = sql
	clause.sqlVars[name] = vars
}

// 根据传入的 Type 的顺序，构造出最终的 SQL 语句
func (clause *Clause) Build(orders ...Type) (string, []interface{}) {
	var sqls []string
	var vars []interface{}

	for _, order := range orders {
		if sql, ok := clause.sql[order]; ok {
			sqls = append(sqls, sql)
			vars = append(vars, clause.sqlVars[order]...)
		}
	}

	return strings.Join(sqls, " "), vars
}
