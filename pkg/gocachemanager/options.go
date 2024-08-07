package gocachemanager

// CacheSettings contains the configuration for building a cache manager.
type CacheSettings struct {
	// RedisConnection is the connection string for the Redis server.
	// Defaults to empty string, meaning no Redis connection. So no cache will be used in Redis.
	redisConnection string

	// skipInMemoryCache is a flag to skip the in-memory cache and utilize redis only.
	// Defaults to false, meaning in-memory cache is used.
	skipInMemoryCache bool

	// prometheusPrefix will be used whenever sending cache metrics to Prometheus.
	prometheusPrefix string
}

// DefaultCacheSettings returns the default cache settings.
func DefaultCacheSettings() *CacheSettings {
	return &CacheSettings{
		redisConnection:   "", // No Redis connection by default
		skipInMemoryCache: false,
	}
}

// CacheOption is an interface for applying cache settings.
type CacheOption func(*CacheSettings)

// WithRedisConnection is a cache option for setting the Redis connection string.
func WithRedisConnection(redisConnection string) CacheOption {
	return func(settings *CacheSettings) {
		settings.redisConnection = redisConnection
	}
}

// WithSkipInMemoryCache is a cache option for skipping the in-memory cache.
func WithSkipInMemoryCache() CacheOption {
	return func(settings *CacheSettings) {
		settings.skipInMemoryCache = true
	}
}

// WithPrometheusPrefix is a cache option for setting the Prometheus prefix.
func WithPrometheusPrefix(prometheusPrefix string) CacheOption {
	return func(settings *CacheSettings) {
		settings.prometheusPrefix = prometheusPrefix
	}
}