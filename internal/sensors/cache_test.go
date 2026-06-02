package sensors

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCache_ServesFromCacheWithinTTL(t *testing.T) {
	var calls int
	c := newCache(time.Minute, func() ([]string, error) {
		calls++
		return []string{"frog-01"}, nil
	})
	now := time.Unix(0, 0)
	c.now = func() time.Time { return now }

	ids, err := c.get()
	require.NoError(t, err)
	require.Equal(t, []string{"frog-01"}, ids)

	now = now.Add(30 * time.Second) // still within the 1m TTL
	ids, err = c.get()
	require.NoError(t, err)
	require.Equal(t, []string{"frog-01"}, ids)
	require.Equal(t, 1, calls, "second call within TTL must not refetch")
}

func TestCache_RefetchesAfterTTL(t *testing.T) {
	results := [][]string{{"a"}, {"b"}}
	var calls int
	c := newCache(time.Minute, func() ([]string, error) {
		r := results[calls]
		calls++
		return r, nil
	})
	now := time.Unix(0, 0)
	c.now = func() time.Time { return now }

	ids, err := c.get()
	require.NoError(t, err)
	require.Equal(t, []string{"a"}, ids)

	now = now.Add(90 * time.Second) // past the 1m TTL
	ids, err = c.get()
	require.NoError(t, err)
	require.Equal(t, []string{"b"}, ids)
	require.Equal(t, 2, calls)
}

func TestCache_DoesNotCacheErrors(t *testing.T) {
	boom := errors.New("boom")
	var calls int
	c := newCache(time.Minute, func() ([]string, error) {
		calls++
		return nil, boom
	})
	now := time.Unix(0, 0)
	c.now = func() time.Time { return now }

	_, err := c.get()
	require.ErrorIs(t, err, boom)

	_, err = c.get()
	require.ErrorIs(t, err, boom)
	require.Equal(t, 2, calls, "errors must not be cached; each call retries")
}

func TestCache_SerializesConcurrentFetches(t *testing.T) {
	var mu sync.Mutex
	var calls int
	release := make(chan struct{})
	c := newCache(time.Minute, func() ([]string, error) {
		mu.Lock()
		calls++
		mu.Unlock()
		<-release // hold the in-flight fetch open until all goroutines are racing
		return []string{"frog"}, nil
	})
	now := time.Unix(0, 0)
	c.now = func() time.Time { return now }

	const n = 16
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			ids, err := c.get()
			require.NoError(t, err)
			require.Equal(t, []string{"frog"}, ids)
		}()
	}
	close(release)
	wg.Wait()

	require.Equal(t, 1, calls, "a burst of concurrent requests must trigger only one fetch")
}

func TestCacheTTL_DefaultWhenUnset(t *testing.T) {
	t.Setenv("SENSORS_CACHE_TTL", "")
	require.Equal(t, defaultCacheTTL, cacheTTL())
}

func TestCacheTTL_ParsesEnv(t *testing.T) {
	t.Setenv("SENSORS_CACHE_TTL", "10m")
	require.Equal(t, 10*time.Minute, cacheTTL())
}

func TestCacheTTL_FallsBackOnInvalidOrNonPositive(t *testing.T) {
	t.Setenv("SENSORS_CACHE_TTL", "nonsense")
	require.Equal(t, defaultCacheTTL, cacheTTL())

	t.Setenv("SENSORS_CACHE_TTL", "0s")
	require.Equal(t, defaultCacheTTL, cacheTTL())
}
