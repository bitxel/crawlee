package main

import (
	"github.com/bitxel/crawlee/crawlee"
	"log"
	"net/http"
	_ "net/http/pprof"
)

func main() {
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	err := crawlee.InitConfig("conf/shopee.yml")
	if err != nil {
		return
	}
	crawlee.Start()
}
