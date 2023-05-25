package geeorm

import (
	"database/sql"
	"geeorm/dialect"
	"geeorm/log"
	"geeorm/session"
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
