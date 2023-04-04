package GeeCache

import (
	"GeeCache/lru"
	"sync"
)

type cache struct {
	// 考虑是不是可以换成读写锁
	mu         sync.Mutex
	// lru 是缓存存放数据的地方，使用 lru 算法淘汰数据
	lru        *lru.Cache
	// 缓存最大空间
	cacheBytes int64
}

// add 向缓存里面添加数据，实质上是向 cache.lru 队列里面添加数据
func (c *cache) add(key string, value ByteView) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.lru == nil {
		c.lru = lru.New(c.cacheBytes, nil)
	}
	c.lru.Add(key, value)
}

// get 从缓存里获取数据，实质上是从 cache.lru 里获取数据
func (c *cache) get(key string) (ByteView, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lru == nil {
		return ByteView{}, false
	}
	if v, ok := c.lru.Get(key); ok {
		return v.(ByteView), true
	}
	return ByteView{}, false
}
