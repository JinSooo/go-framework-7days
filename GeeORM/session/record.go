package session

import (
	"errors"
	"geeorm/clause"
	"reflect"
)

/* ------------------------------ 实现记录增删查改相关的代码 ----------------------------- */

/**
 * insert
 *
 * 	INSERT INTO table_name(col1, col2, col3, ...) VALUES
 *  (A1, A2, A3, ...),
 *  (B1, B2, B3, ...),
 *  ...
 * 	=====>
 * 	s := geeorm.NewEngine("sqlite3", "gee.db").NewSession()
 * 	u1 := &User{Name: "Tom", Age: 18}
 * 	u2 := &User{Name: "Sam", Age: 25}
 * 	s.Insert(u1, u2, ...)
 *
 * Insert 需要将已经存在的对象的每一个字段的值平铺开来
 */

func (session *Session) Insert(values ...interface{}) (int64, error) {
	recordValues := make([]interface{}, 0)

	for _, val := range values {
		// 根据val对象映射出对应的schema表
		table := session.Model(val).RefTable()
		// 设置insert语句
		session.clause.Set(clause.INSERT, table.Name, table.FiledNames)
		// 对应var数组，将val进行平铺
		recordValues = append(recordValues, table.RecordValues(val))
	}

	// 生成values语句
	session.clause.Set(clause.VALUES, recordValues...)
	sql, vars := session.clause.Build(clause.INSERT, clause.VALUES)
	result, err := session.Raw(sql, vars...).Exec()

	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

/**
 * find
 *
 * s := geeorm.NewEngine("sqlite3", "gee.db").NewSession()
 * var users []User
 * s.Find(&users)
 *
 * Find 需要根据平铺开的字段的值构造出对象
 */

func (session *Session) Find(values interface{}) error {
	destSlice := reflect.Indirect(reflect.ValueOf(values))
	// destSlice.Type().Elem() 获取切片的单个元素的类型
	destType := destSlice.Type().Elem()
	// reflect.New(destType).Elem() 获取切片的单个元素的类型
	table := session.Model(reflect.New(destType).Elem().Interface()).RefTable()

	session.clause.Set(clause.SELECT, table.Name, table.FiledNames)
	sql, vars := session.clause.Build(clause.SELECT, clause.WHERE, clause.ORDERBY, clause.LIMIT)
	rows, err := session.Raw(sql, vars...).QueryRows()

	if err != nil {
		return err
	}

	for rows.Next() {
		dest := reflect.New(destType).Elem()
		var value []interface{}

		// 将 dest 的所有字段平铺开，构造切片 values
		for _, name := range table.FiledNames {
			value = append(value, dest.FieldByName(name).Addr().Interface())
		}

		// rows.Scan 该行记录每一列的值依次赋值给 values 中的每一个字段
		if err := rows.Scan(value...); err != nil {
			return err
		}
		// 将 dest 添加到切片 destSlice 中
		destSlice.Set(reflect.Append(destSlice, dest))
	}

	return rows.Close()
}

// 支持对象模式或平铺模式
func (session *Session) Update(values ...interface{}) (int64, error) {
	// 对象
	m, ok := values[0].(map[string]interface{})
	if !ok {
		// 平铺
		m = make(map[string]interface{})
		for i := 0; i < len(values); i += 2 {
			m[values[i].(string)] = values[i+1]
		}
	}

	session.clause.Set(clause.UPDATE, session.RefTable().Name, m)
	sql, vars := session.clause.Build(clause.UPDATE, clause.WHERE)
	result, err := session.Raw(sql, vars...).Exec()

	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

func (session *Session) Delete() (int64, error) {
	session.clause.Set(clause.DELETE, session.RefTable().Name)
	sql, vars := session.clause.Build(clause.DELETE, clause.WHERE)
	result, err := session.Raw(sql, vars...).Exec()

	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

func (session *Session) Count() (int64, error) {
	session.clause.Set(clause.COUNT, session.RefTable().Name)
	sql, vars := session.clause.Build(clause.COUNT)
	row := session.Raw(sql, vars...).QueryRow()

	var tmp int64
	if err := row.Scan(&tmp); err != nil {
		return 0, err
	}

	return tmp, nil
}

/* -------------------------------- 链式调用的条件子句 ------------------------------- */

func (session *Session) Where(desc string, args ...interface{}) *Session {
	var vars []interface{}
	session.clause.Set(clause.WHERE, append(append(vars, desc), args...)...)
	return session
}

func (session *Session) Limit(num int) *Session {
	session.clause.Set(clause.LIMIT, num)
	return session
}

func (session *Session) OrderBy(desc string) *Session {
	session.clause.Set(clause.ORDERBY, desc)
	return session
}

/* ---------------------------------- 简用方法 ---------------------------------- */

func (session *Session) First(value interface{}) error {
	dest := reflect.Indirect(reflect.ValueOf(value))
	// Elem() 获取指针指向的元素
	// Addr() 返回对象地址的指针值
	destSlice := reflect.New(reflect.SliceOf(dest.Type())).Elem()

	if err := session.Limit(1).Find(destSlice.Addr().Interface()); err != nil {
		return err
	}
	if destSlice.Len() == 0 {
		return errors.New("NOT FOUND")
	}

	dest.Set(destSlice.Index(0))
	return nil
}
