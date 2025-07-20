package server

import (
	"bytes"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// we implement a in memory cache for our system
var cache = make(map[string]CacheEntry)

var cacheMutex = &sync.RWMutex{}

func (entry *CacheEntry) isFresh() bool {
	return time.Now().Before(entry.Expires)
}

func calculateExpiry(headers http.Header, cachedTime time.Time) time.Time {
	cacheControl := headers.Get("Cache-Control")
	if cacheControl != "" {
		for _, directive := range parseCacheControl(cacheControl) {
			if _, v, found := parseDirective(directive, "max-age"); found {
				if maxAge, err := strconv.Atoi(v); err == nil {
					return cachedTime.Add(time.Duration(maxAge) * time.Second)
				}
			}
		}
	}

	expiresHeader := headers.Get("Expires")
	if expiresHeader != "" {
		if expiresTime, err := http.ParseTime(expiresHeader); err == nil {
			return expiresTime
		}
	}

	dateHeader := headers.Get("Date")
	lastModifiedHeader := headers.Get("Last-Modified")

	if dateHeader != "" && lastModifiedHeader != "" {
		date, err1 := http.ParseTime(dateHeader)
		lastModified, err2 := http.ParseTime(lastModifiedHeader)
		if err1 == nil && err2 == nil {
			age := date.Sub(lastModified)
			if age > 0 {
				return cachedTime.Add(time.Duration(float64(age) * 0.10))
			}
		}
	}

	return cachedTime.Add(CacheTTL)
}

func parseCacheControl(header string) [][]byte {
	return bytes.Split(bytes.TrimSpace([]byte(header)), []byte(","))
}

func parseDirective(directive []byte, key string) (string, string, bool) {
	parts := bytes.SplitN(directive, []byte("="), 2)
	if len(parts) == 2 && bytes.Equal(bytes.ToLower(bytes.TrimSpace(parts[0])), bytes.ToLower([]byte(key))) {
		return string(bytes.TrimSpace(parts[0])), string(bytes.TrimSpace(parts[1])), true
	}
	return "", "", false 
}

// clearCache empties the in-memory cache.
func ClearCache() {
	cacheMutex.Lock()
	cache = make(map[string]CacheEntry)
	cacheMutex.Unlock()
	log.Println("Cache cleared successfully.")
}