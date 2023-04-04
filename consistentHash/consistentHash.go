package consistentHash

import (
    "fmt"
    "hash/crc32"
    "sort"
)

type Hash func(data []byte) uint32

// Map 是一致性哈希算法
type Map struct {
	hash     Hash
	// replicas 表示虚拟节点的倍数
	replicas int
	// 哈希环
	keys     []int
	// hashMap 表示虚拟节点和实际节点的映射关系
	hashMap  map[int]string
}

func New(replicas int, fn Hash) *Map {
	m := &Map{
		replicas: replicas,
		hash: fn,
		hashMap: make(map[int]string),
	}
	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE
	}
	return m
}


// Add 添加真实节点的名称
// 每个节点制作成 m.replicas 个虚拟节点
// 名称为 key#1 ...
func (m *Map) Add(keys ...string) {
	for _, key := range keys {
		for i := 0; i < m.replicas; i++ {
			// 虚拟节点的名称为 name#1 / name#2 ...
			// 1name 2name
			hash := int(m.hash([]byte(fmt.Sprintf("%d%s",i, key))))
			// 添加到环上
			m.keys = append(m.keys, hash)
			// 建立虚拟节点和实际节点的映射关系
			m.hashMap[hash] = key
		}
	}
	sort.Ints(m.keys)
}

// Get 获取 key 所属的节点名称
func (m *Map) Get(key string) string {
	if len(m.keys) == 0 {
		return ""
	}

	hash := int(m.hash([]byte(key)))
	idx := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hash
	})
	return m.hashMap[m.keys[idx%len(m.keys)]]
}