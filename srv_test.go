package main

import (
	"context"
	"crypto/tls"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/jecolon/mcache"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	setupOnce sync.Once
	client    mcache.CacheClient
	body      []byte
	e         mcache.Entry
)

func setup() {
	creds := credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})
	c, err := grpc.Dial(":8888", grpc.WithTransportCredentials(creds))
	if err != nil {
		log.Fatalf("grpc.Dial: %v", err)
	}

	client = mcache.NewCacheClient(c)
	body = []byte("Hello World!")
	e = mcache.Entry{
		Key:     "foo.txt",
		Ctype:   "text/plain; charset=utf-8",
		Mtime:   time.Now().UTC().UnixNano(),
		Content: body,
		Size:    int64(len(body)),
	}
}

func BenchmarkDataRaces(b *testing.B) {
	setupOnce.Do(setup)

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ne1, _ := client.Get(context.Background(), &mcache.Entry{Key: "foo.txt"})
			ne2, _ := client.Put(context.Background(), &e)
			ne3, _ := client.Get(context.Background(), &mcache.Entry{Key: "foo.txt"})
			ne4, _ := client.Del(context.Background(), &mcache.Entry{Key: "foo.txt"})
			ne5, _ := client.Get(context.Background(), &mcache.Entry{Key: "foo.txt"})

			_ = ne1
			_ = ne2
			_ = ne3
			_ = ne4
			_ = ne5
		}

	})
}

func BenchmarkReadOnly(b *testing.B) {
	setupOnce.Do(setup)
	client.Put(context.Background(), &e)

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ne1, _ := client.Get(context.Background(), &mcache.Entry{Key: "foo.txt"})
			_ = ne1
		}

	})
}

func BenchmarkWriteOnly(b *testing.B) {
	setupOnce.Do(setup)

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ne1, _ := client.Put(context.Background(), &e)
			ne1, _ = client.Del(context.Background(), &e)
			_ = ne1
		}

	})
}
