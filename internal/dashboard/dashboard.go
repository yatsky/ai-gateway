// Package dashboard provides template rendering and asset helpers for the
// gateway web dashboard.
package dashboard

import (
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	httppprof "net/http/pprof"
	"os"
	"strings"

	"github.com/ferro-labs/ai-gateway/internal/apierror"
	webassets "github.com/ferro-labs/ai-gateway/web"
	"github.com/go-chi/chi/v5"
)

var pageTemplates = make(map[string]*template.Template)
var pageTemplatesZH = make(map[string]*template.Template)

func init() {
	pages := []string{
		"getting-started", "overview", "keys", "logs",
		"providers", "config", "analytics", "playground",
	}
	for _, page := range pages {
		// English templates
		tmpl, err := template.ParseFS(webassets.Assets,
			"templates/layout.html",
			"templates/pages/"+page+".html",
		)
		if err != nil {
			panic("failed to parse template " + page + ": " + err.Error())
		}
		pageTemplates[page] = tmpl

		// Chinese (zh-CN) templates
		tmplZH, err := template.ParseFS(webassets.Assets,
			"templates/layout-zh.html",
			"templates/pages/"+page+".html",
		)
		if err != nil {
			panic("failed to parse zh-CN template " + page + ": " + err.Error())
		}
		pageTemplatesZH[page] = tmplZH
	}
}

// DetectLang returns "zh" if the request prefers Chinese, otherwise "en".
func DetectLang(r *http.Request) string {
	// URL parameter overrides
	if lang := r.URL.Query().Get("lang"); lang != "" {
		if strings.HasPrefix(lang, "zh") {
			return "zh"
		}
		return "en"
	}
	// Accept-Language header
	al := r.Header.Get("Accept-Language")
	if strings.Contains(al, "zh") {
		return "zh"
	}
	return "en"
}

// RenderWebTemplate writes the named page template to w.
func RenderWebTemplate(w http.ResponseWriter, pageName string, data any) error {
	return RenderWebTemplateLang(w, pageName, data, "")
}

// RenderWebTemplateLang writes the named page template to w in the given language.
// If lang is empty, it auto-detects from the request (via w).
func RenderWebTemplateLang(w http.ResponseWriter, pageName string, data any, lang string) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if lang == "" {
		lang = "en" // default; caller should use RenderWebTemplateLang with auto-detect
	}
	var tmpl *template.Template
	var ok bool
	if lang == "zh" {
		tmpl, ok = pageTemplatesZH[pageName]
	} else {
		tmpl, ok = pageTemplates[pageName]
	}
	if !ok {
		return fmt.Errorf("unknown page template: %s", pageName)
	}
	layoutName := "layout.html"
	if lang == "zh" {
		layoutName = "layout-zh.html"
	}
	return tmpl.ExecuteTemplate(w, layoutName, data)
}

// MountPprofRoutes registers /debug/pprof/* routes on r when ENABLE_PPROF is set.
func MountPprofRoutes(r chi.Router) {
	if !pprofEnabled() {
		return
	}

	r.Route("/debug/pprof", func(r chi.Router) {
		r.Get("/", httppprof.Index)
		r.Get("/cmdline", httppprof.Cmdline)
		r.Get("/profile", httppprof.Profile)
		r.Post("/symbol", httppprof.Symbol)
		r.Get("/symbol", httppprof.Symbol)
		r.Get("/trace", httppprof.Trace)
		r.Get("/allocs", httppprof.Handler("allocs").ServeHTTP)
		r.Get("/block", httppprof.Handler("block").ServeHTTP)
		r.Get("/goroutine", httppprof.Handler("goroutine").ServeHTTP)
		r.Get("/heap", httppprof.Handler("heap").ServeHTTP)
		r.Get("/mutex", httppprof.Handler("mutex").ServeHTTP)
		r.Get("/threadcreate", httppprof.Handler("threadcreate").ServeHTTP)
	})
}

func pprofEnabled() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("ENABLE_PPROF")))
	return v == "1" || v == "true" || v == "yes"
}

// ServeLogo writes the embedded logo.png to w.
func ServeLogo(w http.ResponseWriter) {
	data, err := fs.ReadFile(webassets.Assets, "logo.png")
	if err != nil {
		apierror.WriteOpenAI(w, http.StatusNotFound, "logo not found", "not_found_error", "resource_not_found")
		return
	}
	w.Header().Set("Content-Type", "image/png")
	_, _ = w.Write(data)
}
