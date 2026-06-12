package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
)

var startTime time.Time

func main() {
	startTime = time.Now()

	port := getEnv("PORT", "3000")
	configPath := getEnv("CONFIG_FILE_PATH", "data/config.json")

	cm, err := NewConfigManager(configPath)
	if err != nil {
		log.Fatalf("Failed to initialize config manager: %v", err)
	}

	r := router.New()

	// ── Public Routes ────────────────────────────────────────────────────────────
	r.GET("/", func(ctx *fasthttp.RequestCtx) {
		ctx.SetContentType("application/json")
		fmt.Fprintf(ctx, `{"name":"Qoder Go Proxy","version":"3.0.0","dashboard":"/dashboard/"}`)
	})

	r.GET("/health", func(ctx *fasthttp.RequestCtx) {
		ctx.SetStatusCode(200)
		fmt.Fprintf(ctx, "ok")
	})

	// ── OpenAI Routes ────────────────────────────────────────────────────────────
	r.GET("/v1/models", func(ctx *fasthttp.RequestCtx) {
		handleModels(ctx, cm)
	})
	r.POST("/v1/chat/completions", func(ctx *fasthttp.RequestCtx) {
		handleChatCompletions(ctx, cm)
	})

	// ── Dashboard Routes ─────────────────────────────────────────────────────────
	r.GET("/dashboard/", func(ctx *fasthttp.RequestCtx) {
		fasthttp.ServeFile(ctx, "public/index.html")
	})
	r.ServeFiles("/dashboard/static/{filepath:*}", "public")

	// Dashboard APIs
	r.GET("/dashboard/api/config", handleConfig)
	r.GET("/dashboard/api/status", handleStatus)
	r.GET("/dashboard/api/settings", func(ctx *fasthttp.RequestCtx) {
		handleGetSettings(ctx, cm)
	})
	r.POST("/dashboard/api/settings", func(ctx *fasthttp.RequestCtx) {
		handlePostSettings(ctx, cm)
	})
	r.GET("/dashboard/api/models", func(ctx *fasthttp.RequestCtx) {
		handleModels(ctx, cm)
	})

	fmt.Printf("🚀 Qoder Go Proxy starting on :%s\n", port)
	if err := fasthttp.ListenAndServe(":"+port, r.Handler); err != nil {
		log.Fatalf("Error in ListenAndServe: %s", err)
	}
}
