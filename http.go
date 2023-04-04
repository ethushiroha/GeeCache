package GeeCache

import (
	"GeeCache/consistentHash"
	pb "GeeCache/geecachepb"
	"fmt"
	"google.golang.org/protobuf/proto"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
)

const (
	defaultPath     = "/_geecache/"
	defaultReplicas = 50
)

// HttpPool 是一个前置节点，整合后面的 HttpGetter
// 它需要可以添加后置存储节点，维护一个一致性哈希环
// 通过一致性哈希找可能存储数据的节点： HttpGetter
type HttpPool struct {
	// self means basic url
	self string
	// path means the uri
	path string
	mu   sync.Mutex
	// peers 表示一致性哈希，可以添加节点和找到节点存储的机器
	peers       *consistentHash.Map
	HttpGetters map[string]*HttpGetter
}

func NewHttpPool(self string) *HttpPool {
	return &HttpPool{
		self:        self,
		path:        defaultPath,
		mu:          sync.Mutex{},
		HttpGetters: make(map[string]*HttpGetter),
	}
}

// Set
// peer in peers like http://127.0.0.1:8001
func (p *HttpPool) Set(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.peers = consistentHash.New(defaultReplicas, nil)
	// 向一致性哈希环内添加存储节点
	p.peers.Add(peers...)

	for _, peer := range peers {
		p.HttpGetters[peer] = &HttpGetter{baseURL: peer + p.path}
	}
}

// PickPeer 用于需要从远程节点获取数据时，找到可能的存储节点
func (p *HttpPool) PickPeer(key string) (PeerGetter, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	// 从一致性哈希环里面找就近的节点
	if peer := p.peers.Get(key); peer != "" && peer != p.self {
		p.Log("Pick peer %s", peer)
		// 返回节点对应的 getter
		return p.HttpGetters[peer], true
	}
	return nil, false
}

var _ PeerPicker = (*HttpPool)(nil)

// Log 记录日志
func (p *HttpPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s", p.self, fmt.Sprintf(format, v...))
}

// ServeHTTP 首先判断路由是否符合规范 `/basicPath/<groupName>/<key>`
// 判断 group 是否存在
// 从 group 中获取值
func (p *HttpPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, p.path) {
		panic("unexpected path: " + r.URL.Path)
	}
	p.Log("%s %s", r.Method, r.URL.Path)

	// path like /basicPath/<groupName>/<key>
	// parts = [<groupName>, <key>]
	parts := strings.SplitN(r.URL.Path[len(p.path):], "/", 2)
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	groupName := parts[0]
	key := parts[1]

	// 根据组名找组
	group := GetGroup(groupName)
	if group == nil {
		http.Error(w, "no such group: "+groupName, http.StatusNotFound)
		return
	}
	// 从 group 里面找 key 对应的 value
	res, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// 将数据写入 response ， 使用 protobuf 通信，用 proto.Marshal 序列化数据
	//	w.Header().Set("Content-Type", "application/octet-stream")
	body, err := proto.Marshal(&pb.Response{Value: res.ByteSlice()})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(body)
}

// HttpGetter 是客户端
// baseURL 表示远程数据源的地址
type HttpGetter struct {
	baseURL string
}

func (h *HttpGetter) Get(in *pb.Request, out *pb.Response) error {
	url := fmt.Sprintf("%s%s/%s", h.baseURL, in.Group, in.Key)
	res, err := http.Get(url)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("server returns: %d", res.StatusCode)
	}
	bytes, err :=  io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	// 获取到数据进行解码
	if err = proto.Unmarshal(bytes, out); err != nil {
		return err
	}
	return nil
}

// 在编译的时候判断 HttpGetter 是不是 PeerGetter 的实现
var _ PeerGetter = (*HttpGetter)(nil)
