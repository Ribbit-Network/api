package sensors

import (
	"os"
	"sync"
	"time"
)

// defaultCacheTTL is how long a fetched sensor list is served before it is
// refetched. The set of sensors changes slowly while the underlying InfluxDB
// schema query is expensive, so a multi-minute cache cuts query load sharply.
const defaultCacheTTL = 5 * time.Minute

// cache memoizes the result of an expensive fetch for a TTL. It holds its lock
// across the fetch so that a burst of concurrent requests triggers at most one
// underlying query; the rest wait and share the result.
type cache struct {
	ttl   time.Duration
	now   func() time.Time
	fetch func() ([]string, error)

	mu      sync.Mutex
	ids     []string
	expires time.Time
	valid   bool
}

func newCache(ttl time.Duration, fetch func() ([]string, error)) *cache {
	return &cache{ttl: ttl, now: time.Now, fetch: fetch}
}

// get returns the cached IDs if they are still fresh, otherwise it refetches.
// Errors are not cached, so a failed fetch is retried on the next call.
func (c *cache) get() ([]string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.valid && c.now().Before(c.expires) {
		return c.ids, nil
	}

	ids, err := c.fetch()
	if err != nil {
		return nil, err
	}

	c.ids = ids
	c.expires = c.now().Add(c.ttl)
	c.valid = true
	return ids, nil
}

// reset clears the cached value, forcing the next get to refetch. Used in tests.
func (c *cache) reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ids = nil
	c.expires = time.Time{}
	c.valid = false
}

// sensorCache is the process-wide cache backing the /sensors handler. It is
// built lazily on first use so that cacheTTL reads SENSORS_CACHE_TTL after the
// .env file has been loaded, and so it picks up test overrides of fetchSensors.
var (
	sensorCacheOnce sync.Once
	sensorCacheInst *cache
)

func sensorCache() *cache {
	sensorCacheOnce.Do(func() {
		sensorCacheInst = newCache(cacheTTL(), func() ([]string, error) {
			return fetchSensors()
		})
	})
	return sensorCacheInst
}

// cacheTTL resolves the cache lifetime from SENSORS_CACHE_TTL (a Go duration
// string such as "10m"), falling back to defaultCacheTTL when unset or invalid.
func cacheTTL() time.Duration {
	if v := os.Getenv("SENSORS_CACHE_TTL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			return d
		}
	}
	return defaultCacheTTL
}
