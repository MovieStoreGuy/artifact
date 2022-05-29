package main

import (
	"context"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/tilinna/clock"
	"go.uber.org/zap"

	"github.com/MovieStoreGuy/artifact"
)

type Client struct {
	interval time.Duration
	domain   string

	log *zap.Logger
	net *http.Client

	rw        sync.RWMutex
	artifacts map[string]artifact.Artifact
}

type OptionFunc func(c *Client) error

var _ artifact.Client = (*Client)(nil)

func WithLogger(log *zap.Logger) OptionFunc {
	return func(c *Client) error {
		c.log = log
		return nil
	}
}

func WithInterval(d time.Duration) OptionFunc {
	return func(c *Client) error {
		c.interval = d
		return nil
	}
}

func NewClient(domain string, opts ...OptionFunc) (artifact.Client, error) {
	c := &Client{
		domain:    domain,
		interval:  10 * time.Second,
		log:       zap.NewNop(),
		net:       &http.Client{},
		artifacts: make(map[string]artifact.Artifact),
	}

	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}

	return c, nil
}

func (c *Client) MonitorUpdates(ctx context.Context) error {
	t := clock.NewTicker(ctx, c.interval)
	defer t.Stop()

	u, err := url.Parse(c.domain)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
		}

		c.log.Info("Checking for changes")
		c.rw.RLock()
		for endpoint, ref := range c.artifacts {
			domain := *u
			domain.Path = endpoint

			req, err := http.NewRequestWithContext(ctx, http.MethodGet, domain.String(), http.NoBody)
			if err != nil {
				return err
			}
			resp, err := c.net.Do(req)
			if err != nil {
				c.log.Info("Failed to reach update", zap.Error(err), zap.String("url", domain.String()))
				continue
			}
			if code := resp.StatusCode; code < 200 && code >= 300 {
				c.log.Info("Unable to process request")
				continue
			}

			mod, err := time.Parse(time.RFC3339, resp.Header.Get("Modified-At"))
			if err != nil {
				continue
			}

			if !mod.After(ref.ModifiedAt()) {
				c.log.Info("Artifact has no updated", zap.String("endpoint", domain.String()))
				continue
			}

			c.log.Info("Updating reference")

			ref.Lock()
			err = ref.Update(resp.Body)
			ref.Unlock()

			c.log.Info("Updated reference")

			_ = resp.Body.Close()

			if err != nil {
				c.log.Error("Unable to update artifact", zap.Error(err), zap.String("endpoint", domain.String()))
				continue
			}

			ref.NotifyUpdated(ctx)
		}
		c.rw.RUnlock()
	}

}

func (c *Client) Register(ctx context.Context, endpoint string, a artifact.Artifact) error {
	u, err := url.Parse(c.domain)
	if err != nil {
		return err
	}
	u.Path = endpoint

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), http.NoBody)
	if err != nil {
		return err
	}

	resp, err := c.net.Do(req)
	if err != nil {
		return err
	}

	c.log.Info("Running update")
	a.Lock()
	err = a.Update(resp.Body)
	a.Unlock()

	defer resp.Body.Close()

	if err != nil {
		return err
	}

	a.NotifyUpdated(ctx)

	c.rw.Lock()
	defer c.rw.Unlock()
	c.artifacts[endpoint] = a
	return nil
}
