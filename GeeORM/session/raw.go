package session

import (
	"database/sql"
	"geeorm/log"
	"strings"
)

/* --------------------------------- 与数据库交互 --------------------------------- */

/**
 * 封装有 2 个目的:
 * 		一是统一打印日志（包括 执行的SQL 语句和错误日志）。
 * 		二是执行完成后，清空 (s *Session).sql 和 (s *Session).sqlVars 两个变量。
 * 		这样 Session 可以复用，开启一次会话，可以执行多次 SQL。
 */

type Session struct {
	// 数据库实例
	db *sql.DB
	// sql语句及占位符
	sql     strings.Builder
	sqlVars []interface{}
}

func New(db *sql.DB) *Session {
	return &Session{db: db}
}

func (session *Session) Clear() {
	session.sql.Reset()
	session.sqlVars = nil
}

func (session *Session) DB() *sql.DB {
	return session.db
}

// 修改sql语句和占位符
func (session *Session) Raw(sql string, values ...interface{}) *Session {
	session.sql.WriteString(sql)
	session.sql.WriteString(" ")
	session.sqlVars = append(session.sqlVars, values...)

	return session
}

func (session *Session) Exec() (result sql.Result, err error) {
	defer session.Clear()
	log.Info(session.sql.String(), session.sqlVars)

	if result, err = session.DB().Exec(session.sql.String(), session.sqlVars...); err != nil {
		log.Error(err)
	}
	return
}

func (session *Session) QueryRow() *sql.Row {
	defer session.Clear()
	log.Info(session.sql.String(), session.sqlVars)

	return session.DB().QueryRow(session.sql.String(), session.sqlVars...)
}

func (session *Session) QueryRows() (rows *sql.Rows, err error) {
	defer session.Clear()
	log.Info(session.sql.String(), session.sqlVars)

	if rows, err = session.DB().Query(session.sql.String(), session.sqlVars...); err != nil {
		log.Error(err)
	}

	return
}
