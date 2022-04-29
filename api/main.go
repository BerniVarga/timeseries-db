package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	cli "github.com/urfave/cli/v2"

	"github.com/gorilla/mux"

	"sky/api/internal/handler"
	"sky/api/internal/storage/mongodb"
)

var dbAddress, dbName, collectionName string

func main() {
	app := &cli.App{
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "dbAddress",
				Usage:       "The address of the timeseries database",
				Destination: &dbAddress,
				Value:       "mongodb://user:password@localhost:27017/sky",
			},
			&cli.StringFlag{
				Name:        "dbName",
				Usage:       "The name of the timeseries database",
				Destination: &dbName,
				Value:       "sky",
			},
			&cli.StringFlag{
				Name:        "collectionName",
				Usage:       "The name of the timeseries collection",
				Destination: &collectionName,
				Value:       "metrics",
			},
		},
		Action: func(c *cli.Context) error {
			ctx := context.Background()
			store, err := mongodb.NewMongoStorage(ctx, dbAddress, "api", dbName, collectionName)
			if err != nil {
				return err
			}

			return run(ctx, store)
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatalf("run failed: %s", err.Error())
	}
}

func run(ctx context.Context, store handler.Store) error {

	r := createRouter(store)
	errCh := make(chan error, 1)

	log.Print("Starting the server on port 8080")
	srv := &http.Server{
		Addr:         "localhost:8080",
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      middleware(r),
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			errCh <- err
		}
	}()
	defer srv.Shutdown(ctx)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	select {
	case <-sigChan:
		return errors.New("shutting down due to received signal")
	case err := <-errCh:
		return err
	}
}

func createRouter(store handler.Store) *mux.Router {
	hndlr := handler.NewMetricsHandler(store)

	r := mux.NewRouter()
	r.HandleFunc("/metrics", hndlr.GetTimeline).Methods(http.MethodGet)
	r.HandleFunc("/metrics/average", hndlr.GetAverage).Methods(http.MethodGet)
	r.HandleFunc("/metrics/{type}", hndlr.GetTimeline).Methods(http.MethodGet)
	r.HandleFunc("/metrics/{type}/average", hndlr.GetAverage).Methods(http.MethodGet)

	return r
}

func middleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-type", "application/json")
		h.ServeHTTP(w, r)
	})
}
