package server

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)
func fetchAndCache(w http.ResponseWriter, r *http.Request, cacheKey string, existingResp *http.Response) {
	var resp *http.Response
	var err error
	var req *http.Request

	if existingResp != nil {
		resp = existingResp
	} else {
		targetURLParsed, parseErr := url.Parse(TargetURL + r.URL.Path)
		if parseErr != nil {
			log.Printf("Error parsing target URL for %s: %v", r.URL.Path, parseErr)
			http.Error(w, "Bad Gateway", http.StatusBadGateway)
			return
		}

		req, err = http.NewRequest(r.Method, targetURLParsed.String(), r.Body)
		if err != nil {
			log.Printf("Error creating origin request for %s: %v", r.URL.Path, err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		for name, values := range r.Header {
			if name == "Host" || name == "Connection" || name == "Keep-Alive" ||
				name == "Proxy-Authenticate" || name == "Proxy-Authorization" ||
				name == "Te" || name == "Trailers" || name == "Transfer-Encoding" || name == "Upgrade" {
				continue
			}
			for _, value := range values {
				req.Header.Add(name, value)
			}
		}

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err = client.Do(req)
		if err != nil {
			log.Printf("Error fetching from origin for %s: %v", r.URL.Path, err)
			http.Error(w, "Bad Gateway (Origin Unreachable/Timeout)", http.StatusBadGateway)
			return
		}
	}

	defer resp.Body.Close() 

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response body from origin for %s: %v", r.URL.Path, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	cacheable := r.Method == http.MethodGet && resp.StatusCode == http.StatusOK &&
		resp.Header.Get("Cache-Control") != "no-store"

	if cacheable {
		newEntry := CacheEntry{
			Body:         bodyBytes,
			Headers:      resp.Header.Clone(),
			Timestamp:    time.Now(),
			Expires:      calculateExpiry(resp.Header, time.Now()), 
			LastModified: resp.Header.Get("Last-Modified"),
			ETag:         resp.Header.Get("ETag"),
		}

		cacheMutex.Lock() 
		cache[cacheKey] = newEntry
		cacheMutex.Unlock() 
		log.Printf("Cached %s from origin.", cacheKey)
	} else {
		log.Printf("Not caching %s (Method: %s, Status: %d, Cache-Control: %s)",
			cacheKey, r.Method, resp.StatusCode, resp.Header.Get("Cache-Control"))
	}

	for header, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(header, value)
		}
	}
	w.Header().Set("X-Cache-Status", "MISS")
	w.WriteHeader(resp.StatusCode)
	_, err = w.Write(bodyBytes)
	if err != nil {
		log.Printf("Error writing response body to client for %s: %v", cacheKey, err)
	}
}
func ProxyHandler(w http.ResponseWriter, r *http.Request) {
	cacheKey := r.URL.Path

	cacheMutex.RLock() 
	entry, found := cache[cacheKey]
	cacheMutex.RUnlock() 
	if found {
		if r.Header.Get("Cache-Control") == "no-cache" {
			log.Printf("Cache-Control: no-cache received for %s. Revalidating with origin.", cacheKey)
		} else if entry.isFresh() {
			log.Printf("Cache HIT for %s (fresh). Serving directly from cache.", cacheKey)

			for header, values := range entry.Headers {
				for _, value := range values {
					w.Header().Add(header, value)
				}
			}
			w.Header().Set("X-Cache-Status", "HIT")
			w.WriteHeader(http.StatusOK)
			_, err := w.Write(entry.Body)
			if err != nil {
				log.Printf("Error writing cached response body for %s: %v", cacheKey, err)
			}
			return 
		} else {
			log.Printf("Cache HIT for %s (stale). Attempting revalidation.", cacheKey)

			req, err := http.NewRequest(r.Method, TargetURL+r.URL.Path, nil)
			if err != nil {
				log.Printf("Error creating revalidation request for %s: %v", cacheKey, err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			if entry.LastModified != "" {
				req.Header.Set("If-Modified-Since", entry.LastModified)
			}
			if entry.ETag != "" {
				req.Header.Set("If-None-Match", entry.ETag)
			}

			client := &http.Client{Timeout: 10 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				log.Printf("Error during revalidation request for %s: %v", cacheKey, err)
				w.Header().Set("Warning", "111 Revalidation Failed")
				w.Header().Set("X-Cache-Status", "HIT (Stale, Revalidation Failed)")
				for header, values := range entry.Headers {
					for _, value := range values {
						w.Header().Add(header, value)
					}
				}
				w.WriteHeader(http.StatusOK)
				_, err := w.Write(entry.Body)
				if err != nil {
					log.Printf("Error writing stale cached response body for %s: %v", cacheKey, err)
				}
				return
			}
			defer resp.Body.Close() 

			if resp.StatusCode == http.StatusNotModified {
				log.Printf("Revalidation successful for %s. Content not modified.", cacheKey)
				cacheMutex.Lock() 
				entry.Timestamp = time.Now()
				entry.Expires = calculateExpiry(resp.Header, entry.Timestamp) 
				cache[cacheKey] = entry
				cacheMutex.Unlock() 

				w.Header().Set("X-Cache-Status", "HIT")
				for header, values := range entry.Headers {
					for _, value := range values {
						w.Header().Add(header, value)
					}
				}
				w.WriteHeader(http.StatusOK)
				_, err := w.Write(entry.Body)
				if err != nil {
					log.Printf("Error writing revalidated cached response body for %s: %v", cacheKey, err)
				}
				return
			} else {
				log.Printf("Revalidation for %s resulted in new content (status: %d). Fetching and updating cache.", cacheKey, resp.StatusCode)
				fetchAndCache(w, r, cacheKey, resp)
				return
			}
		}
	}

	log.Printf("Cache MISS for %s. Fetching from origin.", cacheKey)
	fetchAndCache(w, r, cacheKey, nil)
}



func ShutdownHandler(cancel context.CancelFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("Received shutdown request via /shutdown endpoint. Initiating graceful shutdown...")
		fmt.Fprintf(w, "Shutting down proxy server...\n")
		cancel() 
	}
}
