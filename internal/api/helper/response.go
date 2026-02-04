package helper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"text/template"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zgsm-ai/chat-rag/internal/logger"
	"github.com/zgsm-ai/chat-rag/internal/types"
	"go.uber.org/zap"
)

// SetSSEResponseHeaders sets SSE response headers
func SetSSEResponseHeaders(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream; charset=utf-8")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Headers", "Cache-Control")
	c.Header("X-Accel-Buffering", "no")
}

// SendErrorResponse sends a structured error response
func SendErrorResponse(c *gin.Context, statusCode int, err error) {
	fmt.Printf("==> sendErrorResponse: %+v\n", err)
	message := err.Error()
	errType := "server_error"

	// Check if the error is an APIError with a specific status code
	if apiErr, ok := err.(*types.APIError); ok {
		statusCode = apiErr.StatusCode
		message = apiErr.Message
		errType = apiErr.Type
	}

	c.JSON(statusCode, gin.H{
		"error": map[string]interface{}{
			"message": message,
			"type":    errType,
		},
	})
}

// SendSSEResponseMessage sends a message using SSE format with template rendering
func SendSSEResponseMessage(c *gin.Context, clientIDE string, templateString string, templateData map[string]interface{}) {
	SetSSEResponseHeaders(c)
	c.Status(http.StatusOK)

	// Parse and execute template
	if clientIDE == "vscode" {
		templateString = fmt.Sprintf("{\"result\": \"%s\"}", templateString)
	}
	tmpl, err := template.New("sse").Parse(templateString)

	var responseData string
	if err != nil {
		logger.Error("Failed to parse SSE template", zap.Error(err))
		responseData = templateString
	} else {
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, templateData); err != nil {
			logger.Error("Failed to execute SSE template", zap.Error(err))
			responseData = templateString
		} else {
			responseData = buf.String()
		}
	}

	runes := []rune(responseData)
	flusher, ok := c.Writer.(http.Flusher)

	for i := 0; i < len(runes); i += 2 {
		chunkSize := 2
		if i+chunkSize > len(runes) {
			chunkSize = len(runes) - i
		}
		chunk := string(runes[i : i+chunkSize])

		var response types.ChatCompletionResponse
		if clientIDE == "vscode" {
			response = types.ChatCompletionResponse{
				Id:      "5bf03b8ccd1a4824bffbf36be0a44a78",
				Object:  "chat.completion.chunk",
				Created: time.Now().Unix(),
				Model:   "",
				Choices: []types.Choice{
					{
						Index: 0,
						Delta: types.Delta{
							Role: "assistant",
							ToolCalls: []any{
								map[string]interface{}{
									"id":   "5bf03b8ccd1a4824bffbf36be0a44a78",
									"type": "function",
									"function": map[string]interface{}{
										"name":      "attempt_completion",
										"arguments": chunk,
									},
								},
							},
						},
					},
				},
			}
		} else {
			response = types.ChatCompletionResponse{
				Id:      "",
				Object:  "chat.completion",
				Created: time.Now().Unix(),
				Model:   "",
				Choices: []types.Choice{
					{
						Index: 0,
						Delta: types.Delta{
							Role:    "assistant",
							Content: chunk,
						},
					},
				},
			}
		}

		jsonData, err := json.Marshal(response)
		if err != nil {
			logger.Error("Failed to marshal ChatCompletionResponse", zap.Error(err))
			_, err = fmt.Fprintf(c.Writer, "data: %s\n\n", chunk)
		} else {
			_, err = fmt.Fprintf(c.Writer, "data: %s\n\n", jsonData)
		}
		if err != nil {
			logger.Error("Failed to write SSE response", zap.Error(err))
		}

		if ok {
			flusher.Flush()
		}

		time.Sleep(10 * time.Millisecond)
	}

	c.Writer.Write([]byte("data: [DONE]\n\n"))
	if ok {
		flusher.Flush()
	}
}
