package lru

import "container/list"

/**
 * LRU 算法的实现非常简单，维护一个队列，如果某条记录被访问了，则移动到队尾，那么队首则是最近最少访问的数据，淘汰该条记录即可。
 */

type Cache struct {
	// 允许使用的最大内存
	maxBytes int64
	// 当前已使用的内存
	usedBytes int64
	// 双向队列
	list *list.List
	// 缓存记录映射
	cache map[string]*list.Element
	// 某条记录被移除时的回调函数，可以为 nil
	onCallback callback
}

// 回调函数
type callback func(key string, value Value)

// 双向链表节点的数据类型
type entry struct {
	key   string
	value Value
}

// 为了通用性，允许值是实现了 Value 接口的任意类型
type Value interface {
	// 返回值所占用的内存大小
	Len() int
}

// 实例化
func New(maxBytes int64, onCallback callback) *Cache {
	return &Cache{
		maxBytes:   maxBytes,
		list:       list.New(),
		cache:      make(map[string]*list.Element),
		onCallback: onCallback,
	}
}

// 查找
func (c *Cache) Get(key string) (value Value, ok bool) {
	if ele, ok := c.cache[key]; ok {
		// 移动记录到队尾（双向链表作为队列，队首队尾是相对的，在这里约定 front 为队尾）
		c.list.MoveToFront(ele)
		// 类型断言
		kv := ele.Value.(*entry)
		// 返回查找的值
		return kv.value, true
	}
	return
}

// 删除
func (c *Cache) RemoveOldest() {
	// 拿到队头元素
	ele := c.list.Back()

	if ele != nil {
		// 删除队列中的元素
		c.list.Remove(ele)
		kv := ele.Value.(*entry)
		// 删除缓存中的记录
		delete(c.cache, kv.key)

		// 更新当前所用的内存
		c.usedBytes -= int64(kv.value.Len()) + int64(len(kv.key))
		// 执行回调
		if c.onCallback != nil {
			c.onCallback(kv.key, kv.value)
		}
	}
}

// 添加/修改
func (c *Cache) Set(key string, value Value) {
	// 键已经存在
	if ele, ok := c.cache[key]; ok {
		kv := ele.Value.(*entry)
		// 更新
		c.usedBytes += int64(value.Len()) - int64(kv.value.Len())
		kv.value = value
		// 键不存在
	} else {
		// 加入队尾
		ele := c.list.PushFront(&entry{key: key, value: value})
		c.cache[key] = ele
		c.usedBytes += int64(value.Len()) + int64(len(key))
	}

	// 容量不足，删除队尾
	for c.maxBytes != 0 && c.maxBytes < c.usedBytes {
		c.RemoveOldest()
	}
}

// 缓存记录数量
func (c *Cache) Len() int {
	return c.list.Len()
}