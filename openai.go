package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/valyala/fasthttp"
)

type ChatRequest struct {
	Model           string           `json:"model"`
	Messages        []Message        `json:"messages"`
	Stream          bool             `json:"stream"`
	MaxTokens       int              `json:"max_tokens"`
	ReasoningEffort string           `json:"reasoning_effort"`
}

type Message struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

type ChatChunk struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			Role    string `json:"role,omitempty"`
			Content string `json:"content,omitempty"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
}

func handleChatCompletions(ctx *fasthttp.RequestCtx, cm *ConfigManager) {
	var req ChatRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		ctx.Error("Invalid JSON", http.StatusBadRequest)
		return
	}

	prompt := messagesToPrompt(req.Messages)
	opts := SpawnOptions{
		Model:           req.Model,
		ReasoningEffort: req.ReasoningEffort,
		MaxTokens:       req.MaxTokens,
	}

	stdout, stderrPipe, err := spawnQoderCli(ctx, prompt, opts, cm)
	if err != nil {
		ctx.Error(fmt.Sprintf("Spawn failed: %v", err), http.StatusInternalServerError)
		return
	}
	defer stdout.Close()
	defer stderrPipe.Close()

	id := fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano())
	created := time.Now().Unix()

	if !req.Stream {
		// Non-streaming: accumulate all content
		scanner := bufio.NewScanner(stdout)
		var fullContent strings.Builder
		for scanner.Scan() {
			var line map[string]interface{}
			if err := json.Unmarshal(scanner.Bytes(), &line); err == nil {
				if msg, ok := line["message"].(map[string]interface{}); ok {
					if content, ok := msg["content"].(string); ok {
						fullContent.WriteString(content)
					}
				}
			}
		}
		
		resp := map[string]interface{}{
			"id":      id,
			"object":  "chat.completion",
			"created": created,
			"model":   req.Model,
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": fullContent.String(),
					},
					"finish_reason": "stop",
				},
			},
		}
		json.NewEncoder(ctx).Encode(resp)
		return
	}

	// Streaming: SSE
	ctx.SetContentType("text/event-stream")
	ctx.Response.Header.Set("Cache-Control", "no-cache")
	ctx.Response.Header.Set("Connection", "keep-alive")
	ctx.Response.Header.Set("Transfer-Encoding", "chunked")

	ctx.SetBodyStreamWriter(func(w *bufio.Writer) {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			var line map[string]interface{}
			if err := json.Unmarshal(scanner.Bytes(), &line); err == nil {
				if line["type"] == "assistant" {
					if msg, ok := line["message"].(map[string]interface{}); ok {
						if content, ok := msg["content"].(string); ok {
							chunk := ChatChunk{
								ID: id, Object: "chat.completion.chunk", Created: created, Model: req.Model,
							}
							chunk.Choices = []struct {
								Index int `json:"index"`
								Delta struct {
									Role    string `json:"role,omitempty"`
									Content string `json:"content,omitempty"`
								} `json:"delta"`
								FinishReason *string `json:"finish_reason"`
							}{{Index: 0}}
							chunk.Choices[0].Delta.Content = content
							
							data, _ := json.Marshal(chunk)
							fmt.Fprintf(w, "data: %s\n\n", data)
							w.Flush()
						}
					}
				}
			}
		}
		fmt.Fprintf(w, "data: [DONE]\n\n")
		w.Flush()
	})
}

func handleModels(ctx *fasthttp.RequestCtx, cm *ConfigManager) {
	cfg := cm.Get()
	data := make([]map[string]interface{}, 0, len(cfg.Models))
	for _, m := range cfg.Models {
		data = append(data, map[string]interface{}{
			"id":       m.ID,
			"object":   "model",
			"created":  1677610602,
			"owned_by": "qoder",
		})
	}
	resp := map[string]interface{}{
		"object": "list",
		"data":   data,
	}
	json.NewEncoder(ctx).Encode(resp)
}

func messagesToPrompt(messages []Message) string {
	var sb strings.Builder
	for _, m := range messages {
		content := ""
		switch v := m.Content.(type) {
		case string:
			content = v
		case []interface{}:
			for _, part := range v {
				if p, ok := part.(map[string]interface{}); ok {
					if t, ok := p["text"].(string); ok {
						content += t
					}
				}
			}
		}
		if content != "" {
			sb.WriteString(fmt.Sprintf("%s: %s\n\n", strings.Title(m.Role), content))
		}
	}
	return sb.String()
}
