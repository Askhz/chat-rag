package utils

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"path"
	"strconv"
	"strings"

	"github.com/pkoukk/tiktoken-go"
	"github.com/zgsm-ai/chat-rag/internal/types"
	"github.com/zgsm-ai/chat-rag/internal/utils/assets"
)

// TokenCounter provides token counting functionality
type TokenCounter struct {
	encoder *tiktoken.Tiktoken
}

type OfflineLoader struct{}

func (l *OfflineLoader) LoadTiktokenBpe(tiktokenBpeFile string) (map[string]int, error) {
	baseFileName := path.Base(tiktokenBpeFile)
	contents, err := assets.Assets.ReadFile(baseFileName)
	if err != nil {
		return nil, err
	}

	bpeRanks := make(map[string]int)
	for _, line := range strings.Split(string(contents), "\n") {
		if line == "" {
			continue
		}
		parts := strings.Split(line, " ")
		token, err := base64.StdEncoding.DecodeString(parts[0])
		if err != nil {
			return nil, err
		}
		rank, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, err
		}
		bpeRanks[string(token)] = rank
	}
	return bpeRanks, nil
}

func NewOfflineLoader() *OfflineLoader {
	return &OfflineLoader{}
}

// NewTokenCounter creates a new token counter instance
func NewTokenCounter() (*TokenCounter, error) {
	// Set offline loader to use local encoding files
	loader := NewOfflineLoader()
	tiktoken.SetBpeLoader(loader)

	encoder, err := tiktoken.GetEncoding("cl100k_base")
	if err != nil {
		log.Printf("Failed to initialize tiktoken encoder: %v", err)
		// Return instance with nil encoder which will use fallback estimation
		return &TokenCounter{encoder: nil}, nil
	}

	return &TokenCounter{
		encoder: encoder,
	}, nil
}

// CountTokens counts tokens in a text string
func (tc *TokenCounter) CountTokens(text string) int {
	if tc.encoder == nil {
		log.Printf("[CountTokens][err] Encoder is not initialized")
		// Fallback to simple estimation if encoder is not available
		return len(strings.Fields(text)) * 4 / 3 // Rough approximation
	}

	tokens := tc.encoder.Encode(text, nil, nil)
	return len(tokens)
}

func (tc *TokenCounter) CountMessagesTokens(messages []types.Message) int {
	totalTokens := 0

	for _, message := range messages {
		// Count tokens for role
		totalTokens += tc.CountTokens(message.Role)

		// Count tokens for content
		totalTokens += tc.CountTokens(GetContentAsString(message.Content))

		// Add overhead tokens per message (approximately 3 tokens per message)
		totalTokens += 3
	}

	// Add overhead tokens for the conversation (approximately 3 tokens)
	totalTokens += 3
	return totalTokens
}

func (tc *TokenCounter) CountOneMesaageTokens(message types.Message) int {
	totalTokens := 0

	// Count tokens for role
	totalTokens += tc.CountTokens(message.Role)

	// Count tokens for content
	totalTokens += tc.CountTokens(GetContentAsString(message.Content))

	// Add overhead tokens per message (approximately 3 tokens per message)
	totalTokens += 3

	return totalTokens
}

// CountJSONTokens counts tokens in a JSON object
func (tc *TokenCounter) CountJSONTokens(data interface{}) int {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return 0
	}

	return tc.CountTokens(string(jsonBytes))
}

// EstimateTokens provides a simple token estimation without tiktoken
func EstimateTokens(text string) int {
	// Simple estimation: roughly 4 characters per token
	return len(text) / 4
}
