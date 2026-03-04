package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/joho/godotenv"
)


var (
	db *sql.DB
	c *config
)

func init() {
	_ = godotenv.Load()
	c = new(config{
		host: getEnv("SERVER_HOST", "locahost"),
		port: getEnv("SERVER_PORT", "8080"),
		rto: getEnvAsTime("READ_TIMEOUT", 5 * time.Second),
		wto: getEnvAsTime("WRITE_TIMEOUT", 5 * time.Second),
		ito: getEnvAsTime("IDLE_TIMEOUT", 10 * time.Second),
		sto: getEnvAsTime("SHUTDOWN_TIMEOUT", 10 * time.Second),
		dbUrl: getEnv("DATABASE_URL", "postgres://localhost:5432/shorter?sslmode=disable"),
	})
	db = connect(c.dbUrl)
}

type config struct {
	host string
	port string
	rto, wto, ito, sto time.Duration
	dbUrl string
}

func main() {
	log.Fatal(start(context.Background(), c))
}


func start(ctx context.Context, c *config) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/shorten", func(w http.ResponseWriter, r *http.Request) {
		url := r.FormValue("url")
		if url == "" {
			http.Error(w, "url is required", http.StatusBadRequest)
			return
		}

		shortUrl, err := shortenUrl(ctx, url)
		if err != nil {
			http.Error(w, "Failed to shorten URL", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, shortUrl)
	})
	mux.HandleFunc("/resolve", func(w http.ResponseWriter, r *http.Request) {
		id := r.FormValue("id")
		if id == "" {
			http.Error(w, "id is required", http.StatusBadRequest)
			return
		}

		url, err := resolveUrl(ctx, shortUrl{ID: id})
		if err != nil {
			http.Error(w, "Failed to resolve URL", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, *url, http.StatusMovedPermanently)
	})
	s := new(http.Server{
		Addr: c.host + ":" + c.port,
		ReadTimeout: c.rto,
		IdleTimeout: c.ito,
		WriteTimeout: c.wto,
	})

	serverErr := make(chan error, 1)

	go func() {
		if err := s.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	stopCtx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()
	select {
	case err := <-serverErr:
		return err
	case <-stopCtx.Done():
		ctx, cancel := context.WithTimeout(context.Background(), c.sto)
		defer cancel()
		return s.Shutdown(ctx)
	}
}

func connect(dbUrl string) *sql.DB {
	db, err := sql.Open("postgres", dbUrl)
	if err != nil {
		log.Fatal(err)
	}

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	return db
}



func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvAsTime(key string, fallback time.Duration) time.Duration {
	valueStr, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}

	value, err := time.ParseDuration(valueStr)
	if err != nil {
		return fallback
	}

	return value
}

