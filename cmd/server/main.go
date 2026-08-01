package main

import (
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"calebsons_inc/calebsons_go_cloud_native_distributed_task_queue/internal/api"
	"calebsons_inc/calebsons_go_cloud_native_distributed_task_queue/internal/demo"
	"calebsons_inc/calebsons_go_cloud_native_distributed_task_queue/internal/queue"
	"calebsons_inc/calebsons_go_cloud_native_distributed_task_queue/web"
)

func main() {
	workers := envInt("WORKERS", 3)
	port := envString("PORT", "8080")
	seed := envString("SEED_DEMO", "true") == "true"

	q := queue.New(workers)
	defer q.Stop()

	if seed {
		q.SeedDemo()
	}

	mux := http.NewServeMux()
	apiServer := &api.Server{Queue: q}
	apiServer.Routes(mux)

	staticFS, err := fs.Sub(web.Static, "static")
	if err != nil {
		log.Fatal(err)
	}
	fileServer := http.FileServer(http.FS(staticFS))
	mux.Handle("GET /static/", http.StripPrefix("/static/", fileServer))

	serveHTML := func(name string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			data, err := fs.ReadFile(staticFS, name)
			if err != nil {
				http.Error(w, "page unavailable", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write(data)
		}
	}

	mux.HandleFunc("GET /{$}", serveHTML("index.html"))
	mux.HandleFunc("GET /demos/{kind}", func(w http.ResponseWriter, r *http.Request) {
		kind := r.PathValue("kind")
		if !demo.ValidKind(kind) {
			http.NotFound(w, r)
			return
		}
		serveHTML("demo.html")(w, r)
	})

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	go func() {
		fmt.Printf("Calebsons queue dashboard → http://localhost:%s\n", port)
		fmt.Printf("API health                 → http://localhost:%s/health\n", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	_ = server.Close()
	fmt.Println("shutting down")
}

func envString(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return n
}
