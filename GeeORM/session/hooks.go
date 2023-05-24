package session

import (
	"geeorm/log"
	"reflect"
)

const (
	BeforeQuery  = "BeforeQuery"
	AfterQuery   = "AfterQuery"
	BeforeUpdate = "BeforeUpdate"
	AfterUpdate  = "AfterUpdate"
	BeforeDelete = "BeforeDelete"
	AfterDelete  = "AfterDelete"
	BeforeInsert = "BeforeInsert"
	AfterInsert  = "AfterInsert"
)

func (session *Session) CallHook(method string, value interface{}) {
	// 获取对象上的方法
	fn := reflect.ValueOf(session.RefTable().Model).MethodByName(method)
	if value != nil {
		fn = reflect.ValueOf(value).MethodByName(method)
	}

	// 参数 session
	params := []reflect.Value{reflect.ValueOf(session)}

	if fn.IsValid() {
		if v := fn.Call(params); len(v) > 0 {
			if err, ok := v[0].Interface().(error); ok {
				log.Error(err)
			}
		}
	}
}
