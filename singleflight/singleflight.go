package singleflight

import (
    "log"
    "sync"
	"time"
)

// call 代表正在进行中，或已经结束的请求。使用 sync.WaitGroup 锁避免重入
type call struct {
	wg  sync.WaitGroup
	val interface{}
	err error
}

type FN func() (interface{}, error)

// Group 是 singleflight 的主数据结构，管理不同 key 的请求(call)。
type Group struct {
	mu sync.Mutex
	m  map[string]*call
}

func (g *Group) Do(key string, fn FN) (interface{}, error) {
	g.mu.Lock()
	//	defer g.mu.Unlock()

	// 懒加载 Group.m
	if g.m == nil {
		g.m = make(map[string]*call)
	}

	// 如果短期内已经被查询过了
	if c, ok := g.m[key]; ok {
		log.Println("[singleflight] hit")
		g.mu.Unlock()
		// 等待 key 的 fn 执行完毕
		c.wg.Wait()
		return c.val, c.err
	}

	// 没查询过就新建一个查询
	c := new(call)
	c.wg.Add(1)

	g.m[key] = c
	g.mu.Unlock()

	// 调用 fn 查询
	c.val, c.err = fn()
	c.wg.Done()

	// todo: 不理解为什么要删掉 key
	//	g.mu.Lock()
	//	delete(g.m, key)
	//	g.mu.Unlock()

	go func() {
		time.Sleep(time.Second * 10)
		g.Forget(key)
	}()

	return c.val, c.err
}

func (g *Group) Forget(key string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.m, key)
}
