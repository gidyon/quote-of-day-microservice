package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gidyon/docker/go-docker-compose/model"
	"github.com/gomodule/redigo/redis"
	"github.com/gorilla/mux"
)

func indexHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Welcome! Please hit the `/qod` API to get the quote of the day."))
}

func quoteOfTheDayHandler(conn redis.Conn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		currentTime := time.Now()
		date := currentTime.Format("2006-01-02")

		val, err := redis.String(conn.Do("GET", date))
		if err == redis.ErrNil {
			log.Println("Cache miss for date ", date)
			quoteResp, err := getQuoteFromAPI()
			if err != nil {
				w.Write([]byte("Sorry! We could not get the Quote of the Day. Please try again."))
				return
			}
			quote := quoteResp.Contents.Quotes[0].Quote
			conn.Do("SET", date, quote, 24*time.Hour)
			w.Write([]byte(quote))
		} else {
			log.Println("Cache Hit for date ", date)
			w.Write([]byte(val))
		}
	}
}

// Creates a pool of connections to redis server.
func newRedisPool() *redis.Pool {
	return &redis.Pool{
		// Maximum number of idle connections in the pool.
		MaxIdle: 80,
		// max number of connections
		MaxActive: 12000,
		// Dial is an application supplied function for creating and
		// configuring a connection.
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", getEnv("REDIS_URL", "localhost:6379"))
			if err != nil {
				panic(err.Error())
			}
			return c, err
		},
	}
}
func main() {
	// Create Redis Client
	client := newRedisPool()

	_, err := client.Get().Do("PING")
	if err != nil {
		log.Fatal(err)
	}

	// Create Server and Route Handlers
	r := mux.NewRouter()

	r.HandleFunc("/", indexHandler)
	r.HandleFunc("/qod", quoteOfTheDayHandler(client.Get()))

	srv := &http.Server{
		Handler:      r,
		Addr:         ":8080",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Start Server
	go func() {
		log.Println("Starting Server")
		if err := srv.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()

	// Graceful Shutdown
	waitForShutdown(srv)
}

func waitForShutdown(srv *http.Server) {
	interruptChan := make(chan os.Signal, 1)
	signal.Notify(interruptChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Block until we receive our signal.
	<-interruptChan

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	srv.Shutdown(ctx)

	log.Println("Shutting down")
	os.Exit(0)
}

func getQuoteFromAPI() (*model.QuoteResponse, error) {
	apiURL := "http://quotes.rest/qod.json"
	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	log.Println("Quote API Returned: ", resp.StatusCode, http.StatusText(resp.StatusCode))

	if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
		quoteResp := &model.QuoteResponse{}
		json.NewDecoder(resp.Body).Decode(quoteResp)
		return quoteResp, nil
	} else {
		return nil, errors.New("Could not get quote from API")
	}

}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
