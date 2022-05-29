package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/signal"
	"sync"
	"time"

	"go.uber.org/zap"
)

func main() {
	log, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil {
			io.Copy(io.Discard, r.Body)
			r.Body.Close()
		}
		t := time.Now()
		w.Header().Set("Modified-At", t.Format(time.RFC3339))
		json.NewEncoder(w).Encode(map[string][]string{
			"headlines": {"Food is great", "Pineapple on pizza is still up for debate"},
			"links":     {"#/ref/potatot", "#/ref/pizza"},
		})
		log.Info("Accepted request", zap.String("uri", r.RequestURI))
	}))
	defer s.Close()

	ctx, done := signal.NotifyContext(context.Background(), os.Interrupt)
	defer done()

	cl, err := NewClient(s.URL,
		WithLogger(log.Named("update-client")),
		WithInterval(10*time.Second),
	)
	if err != nil {
		log.Panic("Unable to start client", zap.Error(err))
	}

	var wg sync.WaitGroup

	notify := make(chan struct{}, 1)
	defer close(notify)

	wg.Add(1)
	go func(ctx context.Context, notify <-chan struct{}) {
		defer wg.Done()
		for {
			select {
			case <-notify:
				log.Info("Content has been updated")
			case <-ctx.Done():
				return
			}
		}
	}(ctx, notify)

	wg.Add(1)
	go func(ctx context.Context) {
		defer wg.Done()
		if err := cl.MonitorUpdates(ctx); err != nil {
			log.Error("Failed to run update client", zap.Error(err))
		}
	}(ctx)

	content := Content{}

	if err := cl.Register(ctx, "pineapple", &content); err != nil {
		log.Panic("Unable to register type", zap.Error(err))
	}

	if err := content.Register(notify); err != nil {
		log.Panic("Issue trying register artifact", zap.Error(err))
	}

	log.Info("Finished registering component")

	wg.Wait()
	log.Info("Finished")
}
