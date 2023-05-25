package geeorm

import (
	"database/sql"
	"fmt"
	"geeorm/dialect"
	"geeorm/log"
	"geeorm/session"
	"strings"
)

/* ---------------------------------- 与用户交互 --------------------------------- */

/**
 * Session 负责与数据库的交互，那交互前的准备工作（比如连接/测试数据库），交互后的收尾工作（关闭连接）等就交给 Engine 来负责
 */

type Engine struct {
	db      *sql.DB
	dialect dialect.Dialect
}

func NewEngine(driver, source string) (engine *Engine, err error) {
	db, err := sql.Open(driver, source)
	if err != nil {
		log.Error(err)
		return
	}

	// Send a ping to make sure the database connection is alive.
	if err = db.Ping(); err != nil {
		log.Error(err)
		return
	}

	dia, ok := dialect.GetDialect(driver)
	if !ok {
		log.Errorf("dialect %s Not Found", driver)
		return
	}

	engine = &Engine{db: db, dialect: dia}
	log.Info("Connect database success")
	return
}

func (engine *Engine) Close() {
	if err := engine.db.Close(); err != nil {
		log.Error("Fail to close database")
	}
	log.Info("Close database success")
}

// 通过 Engine 实例创建会话，进而与数据库进行交互
func (engine *Engine) NewSession() *session.Session {
	return session.New(engine.db, engine.dialect)
}

type TxFunc func(*session.Session) (interface{}, error)

// 执行事务
func (engine *Engine) Transaction(fn TxFunc) (result interface{}, err error) {
	session := engine.NewSession()

	if err := session.Begin(); err != nil {
		return nil, err
	}

	defer func() {
		// 发生错误，回滚+报错
		// 否则，提交事务
		if p := recover(); p != nil {
			_ = session.Rollback()
			panic(p)
		} else if err != nil {
			_ = session.Rollback()
		} else {
			// 如果事务提交失败，也进行回滚
			defer func() {
				if err != nil {
					_ = session.Rollback()
				}
			}()
			err = session.Commit()
		}
	}()

	return fn(session)
}

/* --------------------------------- Migrate -------------------------------- */
/**
 * 新增字段
 * 		ALTER TABLE table_name ADD COLUMN col_name col_type;
 *
 * 删除字段
 * 		CREATE TABLE new_table AS SELECT col1, col2, ... from old_table
 * 		DROP TABLE old_table
 * 		ALTER TABLE new_table RENAME TO old_table;
 */

// a - b
func difference(a []string, b []string) (diff []string) {
	m := make(map[string]bool)
	for _, v := range b {
		m[v] = true
	}

	for _, v := range a {
		if _, ok := m[v]; !ok {
			diff = append(diff, v)
		}
	}

	return
}

// 表迁移
func (engine *Engine) Migrate(value interface{}) error {
	_, err := engine.Transaction(func(s *session.Session) (result interface{}, err error) {
		// 注意：这边 s.Model(value) 的时候，refTable已经更新成新的Model了
		if !s.Model(value).HasTable() {
			log.Infof("table %s doesn't exist", s.RefTable().Name)
			return nil, s.CreateTable()
		}

		// 新的Model
		table := s.RefTable()
		rows, _ := s.Raw(fmt.Sprintf("SELECT * FROM %s", table.Name)).QueryRows()
		// 老的Model
		columns, _ := rows.Columns()

		// 需要添加和删除的column
		addCols := difference(table.FiledNames, columns)
		delCols := difference(columns, table.FiledNames)
		log.Infof("added cols: %v, deleted cols: %v", addCols, delCols)

		// 新增字段
		for _, col := range addCols {
			field := table.GetField(col)
			sqlStr := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s;", table.Name, field.Name, field.Type)
			if _, err = s.Raw(sqlStr).Exec(); err != nil {
				return
			}
		}

		// 删除字段
		if len(delCols) == 0 {
			return
		}

		tmpTable := "tmp_" + table.Name
		fieldStr := strings.Join(table.FiledNames, ", ")
		s.Raw(fmt.Sprintf("CREATE TABLE %s AS SELECT %s from %s;", tmpTable, fieldStr, table.Name))
		s.Raw(fmt.Sprintf("DROP TABLE %s;", table.Name))
		s.Raw(fmt.Sprintf("ALTER TABLE %s RENAME TO %s;", tmpTable, table.Name))
		_, err = s.Exec()
		return
	})

	return err
}
