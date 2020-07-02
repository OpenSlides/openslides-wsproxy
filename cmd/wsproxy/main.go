package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	wsproxy "github.com/OpenSlides/openslides-wsproxy/internal"
)

func main() {
	listenAddr := getEnv("LISTEN_ADDR", ":9013")

	ws := wsproxy.New(new(os3))

	srv := &http.Server{Addr: listenAddr, Handler: ws}
	defer func() {
		if err := srv.Shutdown(context.Background()); err != nil {
			log.Printf("Error on HTTP server shutdown: %v", err)
		}
	}()

	go func() {
		log.Printf("Listen on %s", listenAddr)
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	waitForShutdown()
}

// waitForShutdown blocks until the service exists.
//
// It listens on SIGINT and SIGTERM. If the signal is received for a second
// time, the process is killed with statuscode 1.
func waitForShutdown() {
	sigint := make(chan os.Signal, 1)
	// syscall.SIGTERM is not pressent on all plattforms. Since the autoupdate
	// service is only run on linux, this is ok. If other plattforms should be
	// supported, os.Interrupt should be used instead.
	signal.Notify(sigint, syscall.SIGINT, syscall.SIGTERM)
	<-sigint
	go func() {
		<-sigint
		os.Exit(1)
	}()
}

// getEnv returns the value of the environment variable env. If it is empty, the
// defaultValue is used.
func getEnv(env, devaultValue string) string {
	value := os.Getenv(env)
	if value == "" {
		return devaultValue
	}
	return value
}

type os3 struct{}

func (o *os3) GetURL(path string) string {
	switch path {
	case "/system/autoupdate":
		host := getEnv("AUTOUPDATE_HOST", "localhost")
		port := getEnv("AUTOUPDATE_PORT", "8002")
		proto := getEnv("AUTOUPDATE_PROTOCOL", "http")
		return proto + "://" + host + ":" + port + path
	}
	return ""
}
