package test

import (
	"GeeCache"
	"fmt"
	"log"
	"net/http"
	"testing"
)

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func TestHttp(t *testing.T) {
	GeeCache.NewGroup("test", 1024, GeeCache.GetterFunc(func(key string) ([]byte, error) {
		log.Println("db hit")
		if ans, ok := db[key]; ok {
			return []byte(ans), nil
		}
		return nil, fmt.Errorf("no such key")
	}))

	addr := "localhost:9999"
	peers := GeeCache.NewHttpPool(addr)
	log.Println("geecache is running at", addr)
	log.Fatal(http.ListenAndServe(addr, peers))
}
