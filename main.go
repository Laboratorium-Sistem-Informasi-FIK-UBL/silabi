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
	Content         map[string]interface{} `json:"content"`
	VisibleSections map[string]interface{} `json:"visible_sections"`
}

type siteCache struct {
	mu        sync.RWMutex
	site      *publicSite
	fetchedAt time.Time
	lastErr   error
}

func (c *siteCache) get() (*publicSite, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.site, c.lastErr
}

func (c *siteCache) set(site *publicSite, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err != nil {
		// Keep serving the last known-good content on a transient fetch
		// failure instead of blanking the site out.
		c.lastErr = err
		return
	}
	c.site = site
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
		site, err := fetchSite(client, apiBase, slug)
		if err != nil {
			log.Printf("fetch site content failed: %v", err)
		}
		cache.set(site, err)
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
		site, fetchErr := cache.get()
		if site == nil {
			log.Printf("no cached content available (last error: %v)", fetchErr)
			http.Error(w, "Situs sedang tidak dapat diakses, silakan coba lagi sebentar lagi.", http.StatusServiceUnavailable)
			return
		}

		data := make(map[string]interface{}, len(site.Content)+2)
		for k, v := range site.Content {
			data[k] = v
		}
		data["api_base_url"] = apiBase
		data["visible_sections"] = site.VisibleSections

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.Execute(w, data); err != nil {
			log.Printf("template execute error: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
	})

	log.Printf("silabi-web listening on :%s (slug=%q, api=%q, cache_ttl=%s)", port, slug, apiBase, ttl)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func fetchSite(client *http.Client, apiBase, slug string) (*publicSite, error) {
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
	return &site, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
