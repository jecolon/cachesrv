package main

import (
	"context"
	"crypto/tls"
	"log"
	"sync"

	"github.com/jecolon/mcache"
	"google.golang.org/grpc"
)

type cacheServer struct {
	mu    sync.Mutex // protects store
	store map[string]*mcache.Entry
}

func (cs *cacheServer) Get(_ context.Context, e *mcache.Entry) (*mcache.Entry, error) {
	cs.mu.Lock()
	ce, ok := cs.store[e.Key]
	cs.mu.Unlock()
	if !ok {
		return e, mcache.ErrNotFound
	}
	return ce, nil
}
func (cs *cacheServer) Put(_ context.Context, e *mcache.Entry) (*mcache.Entry, error) {
	e.Stat = mcache.Status_OK
	cs.mu.Lock()
	cs.store[e.Key] = e
	cs.mu.Unlock()
	return e, nil
}
func (cs *cacheServer) Del(_ context.Context, e *mcache.Entry) (*mcache.Entry, error) {
	cs.mu.Lock()
	delete(cs.store, e.Key)
	cs.mu.Unlock()
	return e, nil
}

func main() {
	cer, err := tls.LoadX509KeyPair("tls/dev/cert.pem", "tls/dev/key.pem")
	if err != nil {
		log.Fatalf("tls.LoadX509KeyPair: %v", err)
	}

	config := &tls.Config{Certificates: []tls.Certificate{cer}}
	ln, err := tls.Listen("tcp", ":8888", config)
	if err != nil {
		log.Fatalf("tls.Listen: %v ", err)
	}
	defer ln.Close()

	srv := grpc.NewServer()
	mcache.RegisterCacheServer(srv, &cacheServer{store: make(map[string]*mcache.Entry)})

	log.Fatal(srv.Serve(ln))
}
