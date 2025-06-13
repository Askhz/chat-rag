package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/zgsm-ai/chat-rag/internal/model"
	"github.com/zgsm-ai/chat-rag/internal/service"
	"github.com/zgsm-ai/chat-rag/internal/types"
)

func main() {
	// Create metrics service
	metricsService := service.NewMetricsService()

	// Create a sample chat log
	chatLog := &model.ChatLog{
		RequestID:   "test-request-123",
		Timestamp:   time.Now(),
		ClientID:    "test-client",
		ProjectPath: "/test/project",
		Model:       "gpt-3.5-turbo",
		OriginalTokens: model.TokenStats{
			SystemTokens: 100,
			UserTokens:   200,
			All:          300,
		},
		CompressedTokens: model.TokenStats{
			SystemTokens: 80,
			UserTokens:   150,
			All:          230,
		},
		CompressionRatio:       0.77,
		IsUserPromptCompressed: true,
		CompressionTriggered:   true,
		SemanticLatency:        150,
		SummaryLatency:         200,
		MainModelLatency:       1500,
		TotalLatency:           2000,
		ResponseContent:        "Test response",
		Usage:                  types.Usage{CompletionTokens: 50},
		Category:               "coding",
	}

	// Record metrics
	metricsService.RecordChatLog(chatLog)

	fmt.Println("Metrics recorded successfully!")
	fmt.Println("You can now access metrics at http://localhost:8080/metrics")

	// Start a simple HTTP server to serve metrics
	http.Handle("/metrics", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// This would normally use promhttp.Handler(), but for testing we'll just show a message
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Metrics endpoint is working! In the real application, this would show Prometheus metrics."))
	}))

	fmt.Println("Starting test server on :8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
