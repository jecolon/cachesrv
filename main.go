package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
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

var puerto = flag.String("p", ":8888", "Dirección IP y puerto")
var dev = flag.Bool("d", false, "Modo de desarrollo local")

func main() {
	flag.Parse()

	// Lanzamos el server gRPC
	srv := grpc.NewServer()
	go startServer(srv)

	// Velamos por señal para detener el server
	conxCerradas := make(chan struct{})
	go waitForShutdown(conxCerradas, srv)

	// Esperamos a que el shut down termine al cerrar todas las conexiones.
	<-conxCerradas
	fmt.Println("Shut down del servidor mcache gRPC completado exitosamente.")
}

func startServer(srv *grpc.Server) {
	certFile, keyFile := "tls/prod/cert.pem", "tls/prod/key.pem"
	if *dev {
		certFile, keyFile = "tls/dev/cert.pem", "tls/dev/key.pem"
	}
	cer, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		log.Fatalf("tls.LoadX509KeyPair: %v", err)
	}

	config := &tls.Config{Certificates: []tls.Certificate{cer}}
	ln, err := tls.Listen("tcp", *puerto, config)
	if err != nil {
		log.Fatalf("tls.Listen: %v ", err)
	}
	defer ln.Close()

	// Atamos nuestro server a la interface del paquete mcache generada por protoc.
	mcache.RegisterCacheServer(srv, &cacheServer{store: make(map[string]*mcache.Entry)})

	// Iniciamos el server.
	fmt.Printf("Servidor mcache gRPC en puerto %s listo. CTRL+C para detener.\n", *puerto)
	if err := srv.Serve(ln); err != nil && err != grpc.ErrServerStopped {
		// Error iniciando el Server. Posible conflicto de puerto, permisos, etc.
		log.Printf("Error durante Serve: %v", err)
	}
}

func waitForShutdown(conxCerradas chan struct{}, srv *grpc.Server) {
	// Canal para recibir señal de interrupción.
	sigint := make(chan os.Signal, 1)
	// Escuchamos por una señal de interrupción del OS (SIGINT).
	signal.Notify(sigint, os.Interrupt)
	<-sigint
	// Si llegamos aquí, recibimos la señal, iniciamos shut down.
	// Noten se puede usar un Context para posible límite de tiempo.
	fmt.Println("\nShut down del servidor mcache gRPC iniciado...")
	srv.GracefulStop()
	// Cerramos el canal, señalando conexiones ya cerradas.
	close(conxCerradas)
}
