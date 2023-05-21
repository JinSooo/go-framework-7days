package singleflight

import "sync"

// 正在进行中，或已经结束的请求
type call struct {
	// 避免请求重入
	wg  sync.WaitGroup
	val interface{}
	err error
}

// 管理不同 key 的请求(call)
type Group struct {
	mutex sync.Mutex
	m     map[string]*call
}

// 作用: 针对相同的 key，无论 Do 被调用多少次，函数 fn 都只会被调用一次，等待 fn 调用结束了，返回返回值或错误。
func (group *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	group.mutex.Lock()
	// 延迟初始化
	if group.m == nil {
		group.m = make(map[string]*call)
	}

	// 已经存在请求
	if ca, ok := group.m[key]; ok {
		// 解锁并等待请求返回
		group.mutex.Unlock()
		ca.wg.Wait()
		return ca.val, ca.err
	}

	// 不存在请求，新建一个call
	ca := new(call)
	// 发起请求前加锁，让其余重复请求等待，即对于相同的请求，只发送一个请求
	ca.wg.Add(1)
	// 添加到 g.m，表明 key 已经有对应的请求在处理
	group.m[key] = ca
	group.mutex.Unlock()

	// 发起请求
	ca.val, ca.err = fn()
	// 请求完成
	ca.wg.Done()

	group.mutex.Lock()
	// 更新group.m
	delete(group.m, key)
	group.mutex.Unlock()

	return ca.val, ca.err
}
