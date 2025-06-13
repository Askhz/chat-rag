package strategy

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zgsm-ai/chat-rag/internal/types"
)

type mockLLMClient struct {
	GenerateContentFunc func(ctx context.Context, systemPrompt string, messages []types.Message) (string, error)
}

func (m *mockLLMClient) GenerateContent(ctx context.Context, systemPrompt string, messages []types.Message) (string, error) {
	return m.GenerateContentFunc(ctx, systemPrompt, messages)
}

func (m *mockLLMClient) GetModelName() string {
	return "mock-model"
}

func (m *mockLLMClient) ChatLLMWithMessagesRaw(ctx context.Context, messages []types.Message) (types.ChatCompletionResponse, error) {
	return types.ChatCompletionResponse{}, nil
}

func (m *mockLLMClient) ChatLLMWithMessagesStreamRaw(ctx context.Context, messages []types.Message, callback func(string) error) error {
	return nil
}

type mockLLMClient struct {
	GenerateFunc func(context.Context, []types.Message) (string, error)
}

func (m *mockLLMClient) Generate(ctx context.Context, messages []types.Message) (string, error) {
	return m.GenerateFunc(ctx, messages)
}

func TestSummaryProcessor_GenerateUserPromptSummary(t *testing.T) {
	// create mock llm client
	mockLLM := &mockLLMClient{
		GenerateContentFunc: func(ctx context.Context, systemPrompt string, messages []types.Message) (string, error) {
			return "Mocked summary containing Previous Conversation, Current Work and Key Technical Concepts", nil
		},
	}

	// create summary processor
	processor := NewSummaryProcessor("test-splitter", mockLLM)

	// Test cases
	tests := []struct {
		name              string
		semanticContext   string
		messages          []types.Message
		latestUserMessage string
		wantErr           bool
	}{
		{
			name:            "basic conversation summary",
			semanticContext: "Test context",
			messages: []types.Message{
				{
					Role:    "user",
					Content: "Hello, can you help me with something?",
				},
				{
					Role:    "assistant",
					Content: "Of course! What can I help you with?",
				},
			},
			latestUserMessage: "I need help with coding",
			wantErr:           false,
		},
		{
			name:            "with system message",
			semanticContext: "Test context",
			messages: []types.Message{
				{
					Role:    "system",
					Content: "You are a helpful assistant",
				},
				{
					Role:    "user",
					Content: "Hello",
				},
			},
			latestUserMessage: "Test message",
			wantErr:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary, err := processor.GenerateUserPromptSummary(context.Background(), tt.semanticContext, tt.messages)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, summary)
			} else {
				fmt.Println("==> summary:", summary)
				assert.NoError(t, err)
				assert.NotEmpty(t, summary)
				// Verify summary contains key components
				assert.Contains(t, summary, "Previous Conversation")
				assert.Contains(t, summary, "Current Work")
				assert.Contains(t, summary, "Key Technical Concepts")
			}
		})
	}
}
