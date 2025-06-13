package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zgsm-ai/chat-rag/internal/client"
	"github.com/zgsm-ai/chat-rag/internal/config"
	"github.com/zgsm-ai/chat-rag/internal/model"
	"github.com/zgsm-ai/chat-rag/internal/types"
)

// mockHttpClient 模拟HTTP客户端
type mockHttpClient struct {
	handler func(*http.Request) (*http.Response, error)
}

func (m *mockHttpClient) Do(req *http.Request) (*http.Response, error) {
	return m.handler(req)
}

// mockLLMClient 模拟LLM客户端
type mockLLMClient struct {
	classifyResult string
	err            error
}

func (m *mockLLMClient) GenerateContent(ctx context.Context, systemPrompt string, messages []types.Message) (string, error) {
	return m.classifyResult, m.err
}

func (m *mockLLMClient) WithHeaders(headers *http.Header) client.LLMClient {
	return &mockLLMClient{
		classifyResult: m.classifyResult,
		err:            m.err,
	}
}

func TestLoggerService_SanitizeFilename(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		defaultName string
		expected    string
	}{
		{
			name:        "empty string",
			input:       "",
			defaultName: "default",
			expected:    "default",
		},
		{
			name:        "valid name",
			input:       "test",
			defaultName: "default",
			expected:    "test",
		},
		{
			name:        "invalid characters",
			input:       "file\\name:with/invalid*chars?\"",
			defaultName: "default",
			expected:    "filenamewithinvalidchars",
		},
		{
			name:        "too long name",
			input:       strings.Repeat("a", 300),
			defaultName: "default",
			expected:    strings.Repeat("a", 255),
		},
	}

	ls := &LoggerService{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ls.sanitizeFilename(tt.input, tt.defaultName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLoggerService_LogAsync(t *testing.T) {
	tempDir := t.TempDir()

	cfg := config.Config{
		LogFilePath:        tempDir,
		LokiEndpoint:       "http://loki.test",
		LogScanIntervalSec: 10,
		LLMEndpoint:        "http://llm.test",
		ClassifyModel:      "test-model",
	}

	ls := NewLoggerService(cfg)

	// 创建mock LLM客户端
	mockClient := client.NewMockLLMClient("CodeGeneration", nil)
	ls.llmClient = mockClient

	err := ls.Start()
	require.NoError(t, err)
	defer ls.Stop()

	testLog := &model.ChatLog{
		Timestamp: time.Now(),
		Identity: &model.Identity{
			UserName:  "test-user",
			RequestID: "12345",
		},
		CompressedPrompt: []types.Message{
			{
				Role:    types.RoleUser,
				Content: "test prompt",
			},
		},
	}

	ls.LogAsync(testLog, &http.Header{})
	time.Sleep(100 * time.Millisecond)

	files, err := os.ReadDir(ls.tempLogFilePath)
	require.NoError(t, err)
	assert.Greater(t, len(files), 0)
}

func TestLoggerService_UploadToLoki(t *testing.T) {
	tests := []struct {
		name        string
		statusCode  int
		shouldError bool
	}{
		{
			name:        "successful upload",
			statusCode:  http.StatusNoContent,
			shouldError: false,
		},
		{
			name:        "failed upload",
			statusCode:  http.StatusInternalServerError,
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer ts.Close()

			ls := &LoggerService{
				lokiEndpoint: ts.URL,
			}

			testLog := &model.ChatLog{
				Timestamp: time.Now(),
				Identity: &model.Identity{
					UserName:  "test-user",
					RequestID: "12345",
				},
				CompressedPrompt: []types.Message{
					{
						Role:    types.RoleUser,
						Content: "test prompt",
					},
				},
			}

			success := ls.uploadToLoki(testLog)
			assert.Equal(t, !tt.shouldError, success)
		})
	}
}

func TestLoggerService_ClassifyLog(t *testing.T) {
	tests := []struct {
		name           string
		mockResult     string
		mockError      error
		expectedResult string
	}{
		{
			name:           "success classification",
			mockResult:     "CodeGeneration",
			mockError:      nil,
			expectedResult: "CodeGeneration",
		},
		{
			name:           "classification error",
			mockResult:     "",
			mockError:      fmt.Errorf("classification error"),
			expectedResult: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockLLMClient{
				classifyResult: tt.mockResult,
				err:            tt.mockError,
			}

			ls := &LoggerService{
				llmClient: mockClient,
			}

			testLog := &model.ChatLog{
				CompressedPrompt: []types.Message{
					{
						Role:    types.RoleUser,
						Content: "test prompt",
					},
				},
			}

			result := ls.classifyLog(testLog)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestLoggerService_WriteLogToFile(t *testing.T) {
	tempDir := t.TempDir()
	ls := &LoggerService{}

	tests := []struct {
		name      string
		filePath  string
		content   string
		mode      int
		shouldErr bool
	}{
		{
			name:      "write new file",
			filePath:  filepath.Join(tempDir, "test1.log"),
			content:   "test content",
			mode:      os.O_CREATE | os.O_WRONLY,
			shouldErr: false,
		},
		{
			name:      "invalid directory",
			filePath:  filepath.Join("/invalid/path", "test2.log"),
			content:   "test content",
			mode:      os.O_CREATE | os.O_WRONLY,
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ls.writeLogToFile(tt.filePath, tt.content, tt.mode)
			if tt.shouldErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				content, err := os.ReadFile(tt.filePath)
				require.NoError(t, err)
				assert.Equal(t, tt.content+"\n", string(content))
			}
		})
	}
}

func TestLoggerService_ProcessLogs(t *testing.T) {
	tempDir := t.TempDir()
	ls := &LoggerService{
		tempLogFilePath: tempDir,
		logFilePath:     filepath.Join(tempDir, "permanent"),
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()
	ls.lokiEndpoint = ts.URL

	testLog := &model.ChatLog{
		Timestamp: time.Now(),
		Identity: &model.Identity{
			UserName:  "test-user",
			RequestID: "12345",
		},
		CompressedPrompt: []types.Message{
			{
				Role:    types.RoleUser,
				Content: "test prompt",
			},
		},
	}
	logJSON, err := json.Marshal(testLog)
	require.NoError(t, err)

	testFile := filepath.Join(tempDir, "test.log")
	err = os.WriteFile(testFile, logJSON, 0644)
	require.NoError(t, err)

	ls.processLogs()

	permanentFiles, err := os.ReadDir(ls.logFilePath)
	require.NoError(t, err)
	assert.Greater(t, len(permanentFiles), 0)

	_, err = os.Stat(testFile)
	assert.True(t, os.IsNotExist(err))
}

func TestLoggerService_ConcurrentLogging(t *testing.T) {
	tempDir := t.TempDir()
	ls := NewLoggerService(config.Config{
		LogFilePath:        tempDir,
		LogScanIntervalSec: 1,
	})

	err := ls.Start()
	require.NoError(t, err)
	defer ls.Stop()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			log := &model.ChatLog{
				Timestamp: time.Now(),
				Identity: &model.Identity{
					UserName:  fmt.Sprintf("user-%d", i),
					RequestID: fmt.Sprintf("req-%d", i),
				},
				CompressedPrompt: []types.Message{
					{
						Role:    types.RoleUser,
						Content: fmt.Sprintf("test-%d", i),
					},
				},
			}
			ls.LogAsync(log, &http.Header{})
		}(i)
	}

	wg.Wait()
	time.Sleep(200 * time.Millisecond)

	files, err := os.ReadDir(filepath.Join(tempDir, "permanent"))
	require.NoError(t, err)
	assert.Equal(t, 100, len(files))
}
