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
	// Asignamos flags manualmente
	*puerto = "127.0.0.1:8888"
	*dev = true

	// Lanzamos el server
	srv := grpc.NewServer()
	go startServer(srv)

	// Si usamos certificado "self-signed" tenemos que desactivar verificación
	creds := credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})
	c, err := grpc.Dial(*puerto, grpc.WithTransportCredentials(creds))
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

	// Give the server a chance to start
	time.Sleep(3 * time.Second)
}

func equal(ne, e mcache.Entry) bool {
	if ne.Key != e.Key {
		return false
	}
	if ne.Ctype != e.Ctype {
		return false
	}
	if ne.Mtime != e.Mtime {
		return false
	}
	if ne.Size != e.Size {
		return false
	}
	for i := range ne.Content {
		if ne.Content[i] != e.Content[i] {
			return false
		}
	}
	return true
}

func TestPutGet(t *testing.T) {
	setupOnce.Do(setup)
	
	if _, err := client.Put(context.Background(), &e); err != nil {
		t.Fatalf("Error in Put: %v", err)
	}
	ne, err := client.Get(context.Background(), &mcache.Entry{Key: "foo.txt"})
	if err != nil {
		t.Fatalf("Error in Get: %v", err)
	}
	if !equal(*ne, e) {
		t.Fatalf("ne: %v != e: %v", ne, e)
	}
}

func TestDel(t *testing.T) {
	setupOnce.Do(setup)
	
	if _, err := client.Put(context.Background(), &e); err != nil {
		t.Fatalf("Error in Put: %v", err)
	}
	if _, err := client.Del(context.Background(), &mcache.Entry{Key: "foo.txt"}); err != nil {
		t.Fatalf("Error in Del: %v", err)
	}
	ne, err := client.Get(context.Background(), &mcache.Entry{Key: "foo.txt"})
	if err == nil {
		t.Fatalf("No error in Get after Del, got: %v", ne)
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
