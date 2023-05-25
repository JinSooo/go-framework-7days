package session

import "geeorm/log"

// 开启事务
func (session *Session) Begin() (err error) {
	log.Info("transaction begin")
	if session.tx, err = session.db.Begin(); err != nil {
		log.Error(err)
	}
	return
}

// 事务回滚
func (session *Session) Rollback() (err error) {
	log.Info("transaction rollback")
	if err = session.tx.Rollback(); err != nil {
		log.Error(err)
	}
	return
}

// 提交事务
func (session *Session) Commit() (err error) {
	log.Info("transaction commit")
	if err = session.tx.Commit(); err != nil {
		log.Error(err)
	}
	return
}
