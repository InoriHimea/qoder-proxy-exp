package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
)

var startTime time.Time

func main() {
	startTime = time.Now()

	port := getEnv("PORT", "3000")
	configPath := getEnv("CONFIG_FILE_PATH", "data/config.json")
	usagePath := getEnv("USAGE_FILE_PATH", "data/usage.json")

	cm, err := NewConfigManager(configPath)
	if err != nil {
		log.Fatalf("Failed to initialize config manager: %v", err)
	}

	um, err := NewUsageManager(usagePath)
	if err != nil {
		log.Fatalf("Failed to initialize usage manager: %v", err)
	}

	r := router.New()

	// ── Public Routes ────────────────────────────────────────────────────────────
	r.GET("/", func(ctx *fasthttp.RequestCtx) {
		ctx.SetContentType("application/json")
		fmt.Fprintf(ctx, `{"name":"Qoder Go Proxy","version":"3.1.0","dashboard":"/dashboard/"}`)
	})

	r.GET("/health", func(ctx *fasthttp.RequestCtx) {
		ctx.SetStatusCode(200)
		fmt.Fprintf(ctx, "ok")
	})

	// ── OpenAI & Anthropic Routes ───────────────────────────────────────────────
	r.GET("/v1/models", func(ctx *fasthttp.RequestCtx) {
		handleModels(ctx, cm)
	})
	r.POST("/v1/chat/completions", func(ctx *fasthttp.RequestCtx) {
		handleChatCompletions(ctx, cm, um)
	})
	r.POST("/v1/messages", func(ctx *fasthttp.RequestCtx) {
		handleAnthropicMessages(ctx, cm, um)
	})

	// ── Usage API ────────────────────────────────────────────────────────────────
	r.GET("/usage/local", func(ctx *fasthttp.RequestCtx) {
		handleUsageLocal(ctx, um)
	})
	r.POST("/usage/reset-local", func(ctx *fasthttp.RequestCtx) {
		handleUsageReset(ctx, um)
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
	r.GET("/dashboard/api/logs", handleGetRequestLogs)
	r.DELETE("/dashboard/api/logs", handleClearRequestLogs)
	r.GET("/dashboard/api/logs/system", handleGetSystemLogs)
	r.DELETE("/dashboard/api/logs/system", handleClearSystemLogs)

	// Logging Middleware
	handler := func(ctx *fasthttp.RequestCtx) {
		r.Handler(ctx)
		// Only log API requests, skip static files
		path := string(ctx.Path())
		if !strings.HasPrefix(path, "/dashboard/static") && path != "/dashboard/" {
			AddRequestLog(string(ctx.Method()), path, ctx.Response.StatusCode())
		}
	}

	fmt.Printf("🚀 Qoder Go Proxy starting on :%s\n", port)
	AddSystemLog("Qoder Proxy starting...", "info", "system")
	if err := fasthttp.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("Error in ListenAndServe: %s", err)
	}
}
