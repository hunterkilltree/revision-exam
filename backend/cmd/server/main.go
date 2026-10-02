package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"quizlive/internal/httpapi"
	"quizlive/internal/store"
)

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func main() {
	port := env("PORT", "8080")
	base := env("PUBLIC_BASE_URL", "http://localhost:3000")
	origins := strings.Split(env("ALLOWED_ORIGINS", base), ",")
	maxSessions, _ := strconv.Atoi(env("MAX_SESSIONS", "20"))
	ttl, err := time.ParseDuration(env("SESSION_TTL", "2h"))
	if err != nil {
		log.Fatalf("bad SESSION_TTL: %v", err)
	}

	st := store.NewMemory(maxSessions)
	stop := make(chan struct{})
	go st.Janitor(ttl, time.Minute, stop)

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           httpapi.New(httpapi.Config{PublicBaseURL: base, AllowedOrigins: origins}, st),
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		log.Printf("listening on :%s (public %s)", port, base)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	close(stop)
	st.StopAll() // closes sockets with going-away so clients auto-reconnect
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}
