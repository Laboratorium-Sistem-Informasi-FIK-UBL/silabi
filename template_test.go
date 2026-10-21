package main

import (
	"bytes"
	"html/template"
	"testing"
)

func TestParseAndRenderTemplate(t *testing.T) {
	tmpl, err := template.ParseFiles("templates/site.html")
	if err != nil {
		t.Fatalf("failed to parse templates/site.html: %v", err)
	}

	data := map[string]interface{}{
		"company_name": "Laboratorium Sistem Informasi",
		"hero": map[string]interface{}{
			"title":         "Pusat Unggulan Riset & Inovasi",
			"subtitle":      "Mengembangkan talenta dan teknologi.",
			"cta_primary":   "Hubungi Kami",
			"cta_secondary": "Pelajari Riset",
		},
		"about": map[string]interface{}{
			"description": "Lab SI adalah wadah riset.",
			"vision":      "Menjadi laboratorium terdepan.",
			"image_url":   "/assets/lab.jpg",
		},
		"why_us": []map[string]interface{}{
			{"title": "Riset Terapan", "description": "Solusi nyata industri."},
		},
		"stats": map[string]interface{}{
			"founded_year":   "5+",
			"total_projects": 12,
			"total_clients":  8,
			"coverage_area":  "FIK UBL",
		},
		"services": []map[string]interface{}{
			{"name": "Sistem Informasi", "description": "Pengembangan aplikasi.", "link_url": "https://example.com"},
		},
		"team": []map[string]interface{}{
			{"name": "Asep Trisna Setiawan", "role": "Kepala Lab", "bio": "Dosen & Peneliti"},
		},
		"gallery": []map[string]interface{}{
			{"image_url": "/assets/photo.jpg", "caption": "Workshop"},
		},
		"contact": map[string]interface{}{
			"address": "Gedung FIK Kampus UBL",
			"email":   "labsi@ubl.ac.id",
			"phone":   "+62 812-3456-7890",
		},
		"seo": map[string]interface{}{
			"meta_title":       "SILABI - Lab Sistem Informasi FIK UBL",
			"meta_description": "Laboratorium Riset dan Pengembangan Sistem Informasi UBL",
			"og_image_url":     "/assets/og.jpg",
		},
		"footer": map[string]interface{}{
			"year": 2026,
		},
		"site_slug":     "silabi",
		"api_base_url":  "https://api-tangkas.sta.my.id",
		"visible_sections": map[string]interface{}{
			"banner":   true,
			"why_us":   true,
			"stats":    true,
			"about":    true,
			"services": true,
			"team":     true,
			"gallery":  true,
			"contact":  true,
		},
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatalf("failed to execute template: %v", err)
	}

	if buf.Len() == 0 {
		t.Fatalf("rendered output is empty")
	}
}

// TestRenderMinimalData guards the defensive `{{if or (not .visible_sections) ...}}`
// checks throughout the template: visible_sections must default to "show
// everything" not just when present-but-empty, but also when the key is
// missing from the data entirely (e.g. an older cached API response).
func TestRenderMinimalData(t *testing.T) {
	tmpl, err := template.ParseFiles("templates/site.html")
	if err != nil {
		t.Fatalf("failed to parse templates/site.html: %v", err)
	}

	data := map[string]interface{}{
		"company_name": "SILABI",
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatalf("failed to execute template with minimal data: %v", err)
	}
}
