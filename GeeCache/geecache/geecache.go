package geecache

import (
	"fmt"
	"sync"
)

/* -------------------------- 负责与外部交互，控制缓存存储和获取的主流程 ------------------------- */
/**
 * 缓存获取流程：
 *                            是
 * 接收 key --> 检查是否被缓存 -----> 返回缓存值 ⑴
 *                 |  否                         是
 *                 |-----> 是否应当从远程节点获取 -----> 与远程节点交互 --> 返回缓存值 ⑵
 *                             |  否
 *                             |-----> 调用`回调函数`，获取值并添加到缓存 --> 返回缓存值 ⑶
 */

/* --------------------------------- 回调 Getter --------------------------------- */
/**
 * 如果缓存不存在，应从数据源（文件，数据库等）获取数据并添加到缓存中
 * 设计了一个回调函数(callback)，在缓存不存在时，调用这个函数，得到源数据。
 */

/**
 * 定义一个函数类型 F，并且实现接口 A 的方法，然后在这个方法中调用自己。
 * 这是 Go 语言中将其他函数（参数返回值定义与 F 一致）转换为接口 A 的常用技巧。
 */
type Getter interface {
	Get(key string) ([]byte, error)
}

// 用于处理缓存没命中时的本地源数据
type GetterFunc func(key string) ([]byte, error)

func (fn GetterFunc) Get(key string) ([]byte, error) {
	// 借助 GetterFunc 的类型转换，将一个匿名回调函数转换成了接口 f Getter
	return fn(key)
}

/* ---------------------------------- Group --------------------------------- */
/**
 * Group是一个缓存名称空间和相关数据加载分布
 * 一个 Group 可以认为是一个缓存的命名空间，每个 Group 拥有一个唯一的名称 name。
 * 比如可以创建三个 Group，缓存学生的成绩命名为 scores，缓存学生信息的命名为 info，缓存学生课程的命名为 courses。
 */
type Group struct {
	name string
	// 缓存未命中时获取源数据的回调(callback)
	getter Getter
	// 并发缓存
	mainCache cache
}

var (
	// 读写锁
	mutex sync.RWMutex
	// 所有Group的集合
	groups = make(map[string]*Group)
)

// 实例化一个Group
func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter")
	}

	mutex.Lock()
	defer mutex.Unlock()

	group := &Group{
		name:      name,
		getter:    getter,
		mainCache: cache{cacheBytes: cacheBytes},
	}
	groups[name] = group

	return group
}

// 获取一个Group
func GetGroup(name string) *Group {
	mutex.RLock()
	group := groups[name]
	mutex.RUnlock()

	return group
}

/* ---------------------------------- 获取缓存 ---------------------------------- */
func (group *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}

	// 从 mainCache 中查找缓存，如果存在则返回缓存值
	if value, ok := group.mainCache.get(key); ok {
		fmt.Println("[GeeCache] hit")
		return value, nil
	}

	return group.load(key)
}

// 缓存没命中时，加载本地或远程数据源
func (group *Group) load(key string) (ByteView, error) {
	return group.getLocally(key)
}

// getLocally 调用用户回调函数 g.getter.Get() 获取源数据
func (group *Group) getLocally(key string) (ByteView, error) {
	// 源数据
	bytes, err := group.getter.Get(key)

	if err != nil {
		return ByteView{}, err
	}

	// ByteView包装，并缓存
	value := ByteView{b: cloneBytes(bytes)}
	group.populateCache(key, value)

	return value, nil
}

// 将源数据添加到缓存 mainCache 中
func (group *Group) populateCache(key string, value ByteView)  {
	group.mainCache.set(key, value)
}