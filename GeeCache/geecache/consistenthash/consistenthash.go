package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

/* --------------------------------- 一致性哈希代码 -------------------------------- */
/**
 * 一致性哈希算法将 key 映射到 2^32 的空间中，将这个数字首尾相连，形成一个环。
 * 		计算节点/机器(通常使用节点的名称、编号和 IP 地址)的哈希值，放置在环上。
 * 		计算 key 的哈希值，放置在环上，顺时针寻找到的第一个节点，就是应选取的节点/机器。
 *
 * 数据倾斜问题
 * 		引入了虚拟节点的概念，一个真实节点对应多个虚拟节点。
 * 		第一步，计算虚拟节点的 Hash 值，放置在环上。
 * 		第二步，计算 key 的 Hash 值，在环上顺时针寻找到应选取的虚拟节点，例如是 peer2-1，那么就对应真实节点 peer2。
 */

// 哈希映射
type Hash func(data []byte) uint32

type Map struct {
	// 哈希函数
	hash     Hash
	// 一个真实节点对应的虚拟节点的个数
	replicas int
	// 哈希环（有序）
	keys     []int
	// 虚拟节点-真实节点的映射
	hashMap  map[int]string
}

func NewMap(replicas int, hash Hash) *Map {
	m := &Map{
		hash: hash,
		replicas: replicas,
		hashMap: make(map[int]string),
	}

	// 默认哈希算法 crc32.ChecksumIEEE
	if hash == nil {
		m.hash = crc32.ChecksumIEEE
	}

	return m
}

// 添加真实节点/机器
func (m *Map) Add(keys ...string) {
	for _, key := range keys {
		// 生成replicas个虚拟节点
		for i := 0; i < m.replicas; i++ {
			// 哈希值，虚拟节点的名称是：strconv.Itoa(i) + key
			hash := int(m.hash([]byte(strconv.Itoa(i) + key)))
			// 添加到哈希环
			m.keys = append(m.keys, hash)
			// 添加映射关系
			m.hashMap[hash] = key
		}
	}
	// 哈希环排序
	sort.Ints(m.keys)
}

// 根据key获取到最近的真实节点
func (m *Map) Get(key string) string {
	if len(m.keys) == 0 {
		return ""
	}

	hash := int(m.hash([]byte(key)))
	// 二分查找，顺时针找到第一个匹配的虚拟节点的下标 index
	index := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hash
	})

	// 找到真实节点
	return m.hashMap[m.keys[index % len(m.keys)]]
}