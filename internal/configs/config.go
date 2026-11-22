package configs

import (
	"log"
	"os"
	"strings"
	"time"
)

type Config struct {
	natsAddress     string
	etcdAddress     string
	serverAddress   string
	oortAddress     string
	meridianAddress string
	gravityAddress  string
	tokenKey        string
	staleAfter      time.Duration
	offlineAfter    time.Duration
	gcEnabled       bool
	gcInterval      time.Duration
	gcPoolAfter     time.Duration
}

func (c *Config) NatsAddress() string {
	return c.natsAddress
}

func (c *Config) EtcdAddress() string {
	return c.etcdAddress
}

func (c *Config) ServerAddress() string {
	return c.serverAddress
}

func (c *Config) OortAddress() string {
	return c.oortAddress
}

func (c *Config) MeridianAddress() string {
	return c.meridianAddress
}

func (c *Config) GravityAddress() string {
	return c.gravityAddress
}

func (c *Config) TokenKey() string {
	return c.tokenKey
}

func (c *Config) StaleAfter() time.Duration {
	return c.staleAfter
}

func (c *Config) OfflineAfter() time.Duration {
	return c.offlineAfter
}

func (c *Config) GCEnabled() bool {
	return c.gcEnabled
}

func (c *Config) GCInterval() time.Duration {
	return c.gcInterval
}

func (c *Config) GCPoolAfter() time.Duration {
	return c.gcPoolAfter
}

const (
	// A node heartbeats every HEARTBEAT_INTERVAL (default 30s in star), so
	// three missed beats make it stale and ten make it offline.
	defaultStaleAfter   = 90 * time.Second
	defaultOfflineAfter = 5 * time.Minute
	// GC only ever touches unclaimed records; 24h is generous on purpose.
	defaultGCInterval  = 10 * time.Minute
	defaultGCPoolAfter = 24 * time.Hour
)

func NewFromEnv() (*Config, error) {
	staleAfter := durationFromEnv("NODE_STALE_AFTER", defaultStaleAfter)
	offlineAfter := durationFromEnv("NODE_OFFLINE_AFTER", defaultOfflineAfter)
	if offlineAfter < staleAfter {
		log.Printf("NODE_OFFLINE_AFTER (%s) is below NODE_STALE_AFTER (%s); using %s", offlineAfter, staleAfter, staleAfter)
		offlineAfter = staleAfter
	}
	return &Config{
		natsAddress:     os.Getenv("NATS_ADDRESS"),
		etcdAddress:     os.Getenv("ETCD_ADDRESS"),
		serverAddress:   os.Getenv("MAGNETAR_ADDRESS"),
		oortAddress:     os.Getenv("OORT_ADDRESS"),
		meridianAddress: os.Getenv("MERIDIAN_ADDRESS"),
		gravityAddress:  os.Getenv("GRAVITY_ADDRESS"),
		tokenKey:        os.Getenv("SECRET_KEY"),
		staleAfter:      staleAfter,
		offlineAfter:    offlineAfter,
		gcEnabled:       boolFromEnv("NODE_GC_ENABLED", true),
		gcInterval:      durationFromEnv("NODE_GC_INTERVAL", defaultGCInterval),
		gcPoolAfter:     durationFromEnv("NODE_GC_POOL_AFTER", defaultGCPoolAfter),
	}, nil
}

func durationFromEnv(key string, def time.Duration) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return def
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		log.Printf("invalid %s=%q; using %s", key, raw, def)
		return def
	}
	return d
}

func boolFromEnv(key string, def bool) bool {
	raw := os.Getenv(key)
	if raw == "" {
		return def
	}
	switch strings.ToLower(raw) {
	case "1", "true", "yes":
		return true
	case "0", "false", "no":
		return false
	}
	log.Printf("invalid %s=%q; using %v", key, raw, def)
	return def
}
