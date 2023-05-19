package geecache

import (
	"geecache/geecache/lru"
	"sync"
)

/* -------------------------------- 对缓存添加并发控制 ------------------------------- */

type cache struct {
	// 互斥锁
	mutex sync.Mutex
	// lru缓存
	lru *lru.Cache
	// 缓存内存大小
	cacheBytes int64
}

// 添加/修改
func (cache *cache) set(key string, value ByteView) {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()

	// 延迟初始化
	if cache.lru == nil {
		cache.lru = lru.New(cache.cacheBytes, nil)
	}

	cache.lru.Set(key, value)
}

// 查找
func (cache *cache) get(key string) (value ByteView, ok bool) {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()

	if cache.lru == nil {
		return
	}

	if value, ok := cache.lru.Get(key); ok {
		return value.(ByteView), true
	}

	return
}
