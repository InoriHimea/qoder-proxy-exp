package main

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/valyala/fasthttp"
)

type SystemLog struct {
	Timestamp string `json:"timestamp"`
	Message   string `json:"message"`
	Level     string `json:"level"`
	Source    string `json:"source"`
}

type RequestLog struct {
	Timestamp  string `json:"timestamp"`
	Method     string `json:"method"`
	Path       string `json:"path"`
	StatusCode int    `json:"statusCode"`
}

var (
	systemLogs  = make([]SystemLog, 0)
	requestLogs = make([]RequestLog, 0)
	logMu       sync.RWMutex
)

func AddSystemLog(msg, level, source string) {
	logMu.Lock()
	defer logMu.Unlock()
	systemLogs = append(systemLogs, SystemLog{
		Timestamp: time.Now().Format(time.RFC3339),
		Message:   msg,
		Level:     level,
		Source:    source,
	})
	if len(systemLogs) > 500 {
		systemLogs = systemLogs[len(systemLogs)-500:]
	}
}

func AddRequestLog(method, path string, status int) {
	logMu.Lock()
	defer logMu.Unlock()
	requestLogs = append(requestLogs, RequestLog{
		Timestamp:  time.Now().Format(time.RFC3339),
		Method:     method,
		Path:       path,
		StatusCode: status,
	})
	if len(requestLogs) > 500 {
		requestLogs = requestLogs[len(requestLogs)-500:]
	}
}

func handleGetSystemLogs(ctx *fasthttp.RequestCtx) {
	logMu.RLock()
	defer logMu.RUnlock()
	json.NewEncoder(ctx).Encode(map[string]interface{}{"logs": systemLogs})
}

func handleGetRequestLogs(ctx *fasthttp.RequestCtx) {
	logMu.RLock()
	defer logMu.RUnlock()
	json.NewEncoder(ctx).Encode(map[string]interface{}{"logs": requestLogs})
}

func handleClearSystemLogs(ctx *fasthttp.RequestCtx) {
	logMu.Lock()
	systemLogs = make([]SystemLog, 0)
	logMu.Unlock()
	json.NewEncoder(ctx).Encode(map[string]interface{}{"ok": true})
}

func handleClearRequestLogs(ctx *fasthttp.RequestCtx) {
	logMu.Lock()
	requestLogs = make([]RequestLog, 0)
	logMu.Unlock()
	json.NewEncoder(ctx).Encode(map[string]interface{}{"ok": true})
}
