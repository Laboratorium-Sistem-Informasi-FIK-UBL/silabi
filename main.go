// Command silabi-web is a small, dependency-free (standard library only)
// server for silabi.ubl.ac.id. It owns no content of its own: on every
// cache refresh it fetches the published site content for slug SITE_SLUG
// from the Tangkas backend's public API (GET /api/public/site/{slug}) and
// renders it server-side into templates/site.html. Content management
// (hero, team, gallery, contact, dsb.) stays entirely in the Tangkas
// dashboard; this binary is purely a presentation layer on its own domain.
package main

import (
	"encoding/json"
	"errors"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

type envelope struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
	Error   string          `json:"error"`
}

type publicSite struct {
	Content map[string]interface{} `json:"content"`
}

type siteCache struct {
	mu        sync.RWMutex
	content   map[string]interface{}
	fetchedAt time.Time
	lastErr   error
}

func (c *siteCache) get() (map[string]interface{}, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.content, c.lastErr
}

func (c *siteCache) set(content map[string]interface{}, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err != nil {
		// Keep serving the last known-good content on a transient fetch
		// failure instead of blanking the site out.
		c.lastErr = err
		return
	}
	c.content = content
	c.fetchedAt = time.Now()
	c.lastErr = nil
}

func main() {
	apiBase := getEnv("TANGKAS_API_URL", "https://api-tangkas.sta.my.id")
	slug := getEnv("SITE_SLUG", "silabi")
	port := getEnv("PORT", "8746")
	ttl := 60 * time.Second
	if v := os.Getenv("CACHE_TTL_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			ttl = time.Duration(n) * time.Second
		}
	}

	tmpl, err := template.ParseFiles("templates/site.html")
	if err != nil {
		log.Fatalf("failed to parse templates/site.html: %v", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	cache := &siteCache{}

	refresh := func() {
		content, err := fetchSiteContent(client, apiBase, slug)
		if err != nil {
			log.Printf("fetch site content failed: %v", err)
		}
		cache.set(content, err)
	}

	refresh() // warm the cache before accepting traffic
	go func() {
		ticker := time.NewTicker(ttl)
		defer ticker.Stop()
		for range ticker.C {
			refresh()
		}
	}()

	mux := http.NewServeMux()
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("assets"))))
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		content, fetchErr := cache.get()
		if content == nil {
			log.Printf("no cached content available (last error: %v)", fetchErr)
			http.Error(w, "Situs sedang tidak dapat diakses, silakan coba lagi sebentar lagi.", http.StatusServiceUnavailable)
			return
		}

		data := make(map[string]interface{}, len(content)+1)
		for k, v := range content {
			data[k] = v
		}
		data["api_base_url"] = apiBase

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.Execute(w, data); err != nil {
			log.Printf("template execute error: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
	})

	log.Printf("silabi-web listening on :%s (slug=%q, api=%q, cache_ttl=%s)", port, slug, apiBase, ttl)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func fetchSiteContent(client *http.Client, apiBase, slug string) (map[string]interface{}, error) {
	resp, err := client.Get(apiBase + "/api/public/site/" + slug)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var env envelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return nil, err
	}
	if !env.Success {
		if env.Error != "" {
			return nil, errors.New(env.Error)
		}
		return nil, errors.New("tangkas api returned success=false")
	}

	var site publicSite
	if err := json.Unmarshal(env.Data, &site); err != nil {
		return nil, err
	}
	return site.Content, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
