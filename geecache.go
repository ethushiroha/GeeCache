package GeeCache

import (
    pb "GeeCache/geecachepb"
    "GeeCache/singleflight"
    "fmt"
	"log"
	"sync"
)

// Getter 用于在远程查找数据失败时调用本地函数获取值
// 即，分布式缓存A从其他的缓存机器中获取值失败时，从数据库获取值
type Getter interface {
	Get(key string) ([]byte, error)
}

type GetterFunc func(key string) ([]byte, error)

func (get GetterFunc) Get(key string) ([]byte, error) {
	return get(key)
}

type Group struct {
	name string
	// getter 回调函数，从数据库获取值
	getter    Getter
	mainCache cache
	// peer 其他的分布式服务器
	peer PeerPicker

	loader *singleflight.Group
}

var (
	mu     sync.RWMutex
	groups = make(map[string]*Group)
)

func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil getter func")
	}
	mu.Lock()
	defer mu.Unlock()

	group := &Group{
		name:   name,
		getter: getter,
		mainCache: cache{
			mu:         sync.Mutex{},
			lru:        nil,
			cacheBytes: cacheBytes,
		},
		loader: &singleflight.Group{},
	}
	groups[name] = group
	return group
}

func GetGroup(name string) *Group {
	mu.RLock()
	defer mu.RUnlock()
	g := groups[name]
	return g
}

// RegisterPeers 注册一个前置节点 PeerPicker,
func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peer != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.peer = peers
}

// Get 用于获取 key 对应的值
// 优先从 cache 获取，如果没有，则尝试远程加载/回调函数加载
func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}
	// 优先从缓存中读取
	if v, ok := g.mainCache.get(key); ok {
		log.Println("[GeeCache] hit")
		return v, nil
	}
	// 如果没有命中，尝试去加载 key
	return g.load(key)
}

// load 先从远程节点获取值，如果远程节点获取失败，则通过回调函数获取
// 添加 singleflight ，短期内只获取一次
func (g *Group) load(key string) (ByteView, error) {
	// 使用 loader.Do 包裹原来的函数
	view, err := g.loader.Do(key, func() (interface{}, error) {
		if g.peer != nil {
			// 找到远程节点
			if getter, ok := g.peer.PickPeer(key); ok {
				// 从远程节点获取值
				if res, err := g.getFromPeer(getter, key); err == nil {
					// 不需要把数据放在本地 cache，
					// g.popularCache(key, res)
					return res, err
				} else {
					log.Println("[GeeCache] Failed to get from peer", err)
				}
			}
		}
		// 用回调函数获取值
		return g.getLocally(key)
    })

	if err != nil {
		return ByteView{}, err
	}
	return view.(ByteView), nil
}

// getFromPeer 是从远程节点获取值，调用 PeerGetter.Get() 方法
func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	req := &pb.Request{
		Group: g.name,
		Key: key,
	}
	res := &pb.Response{}
	if err := peer.Get(req, res); err != nil {
		return ByteView{}, err
	} else {
		return ByteView{b: res.Value}, nil
	}
}

// getLocally 从回调函数处获取值，并且保存在 cache 中
func (g *Group) getLocally(key string) (ByteView, error) {
	// 调用 getter 方法，从回调函数处获取值
	bytes1, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err
	}
	value := ByteView{b: clone(bytes1)}
	// 将值存放到 cache 中
	g.popularCache(key, value)
	return value, nil
}

// popularCache 将数据保存到 cache 中
func (g *Group) popularCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}
