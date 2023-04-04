# GeeCache

项目来自于：https://geektutu.com/post/geecache.html
感谢作者手把手教学

技术内容：
- LRU淘汰算法
- 并发读写
- http 接口
- 一致性哈希
- 分布式节点
- 缓存击穿 -> singleflight
- protobuf

测试：
1. 运行 test/main_test.go/TestPort8001
2. 运行 test/main_test.go/TestPort8002
3. 运行 test/main_test.go/TestPort8003
4. 访问接口 `http://localhost:9999/api?key=`
