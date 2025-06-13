package utils

import (
	"testing"

	"github.com/zgsm-ai/chat-rag/internal/types"
)

func TestGetContentAsString(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    string
		wantErr bool
	}{
		{
			name:    "empty string",
			input:   "",
			want:    "",
			wantErr: false,
		},
		{
			name:    "simple string",
			input:   "hello",
			want:    "hello",
			wantErr: false,
		},
		{
			name: "content list with text",
			input: []any{
				map[string]any{
					"type": ContentTypeText,
					"text": "hello",
				},
			},
			want:    "hello",
			wantErr: false,
		},
		{
			name:    "empty content list",
			input:   []any{},
			want:    "",
			wantErr: false,
		},
		{
			name:    "invalid content type",
			input:   123,
			want:    "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetContentAsString(tt.input)
			if got != tt.want {
				t.Errorf("GetContentAsString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetUserMsgs(t *testing.T) {
	tests := []struct {
		name   string
		input  []types.Message
		expect []types.Message
	}{
		{
			name:   "empty messages",
			input:  []types.Message{},
			expect: []types.Message{},
		},
		{
			name: "no user messages",
			input: []types.Message{
				{Role: types.RoleSystem, Content: "system"},
			},
			expect: []types.Message{},
		},
		{
			name: "mix user and system messages",
			input: []types.Message{
				{Role: types.RoleSystem, Content: "system"},
				{Role: types.RoleUser, Content: "user 1"},
				{Role: types.RoleUser, Content: "user 2"},
			},
			expect: []types.Message{
				{Role: types.RoleUser, Content: "user 1"},
				{Role: types.RoleUser, Content: "user 2"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetUserMsgs(tt.input)
			if len(got) != len(tt.expect) {
				t.Fatalf("expected %d messages, got %d", len(tt.expect), len(got))
			}
			for i := range got {
				if got[i].Content != tt.expect[i].Content || got[i].Role != tt.expect[i].Role {
					t.Errorf("message %d mismatch: got %v, expected %v", i, got[i], tt.expect[i])
				}
			}
		})
	}
}

func TestGetSystemMsg(t *testing.T) {
	tests := []struct {
		name   string
		input  []types.Message
		expect types.Message
	}{
		{
			name:   "empty messages",
			input:  []types.Message{},
			expect: types.Message{Role: types.RoleSystem, Content: ""},
		},
		{
			name: "only user messages",
			input: []types.Message{
				{Role: types.RoleUser, Content: "user"},
			},
			expect: types.Message{Role: types.RoleSystem, Content: ""},
		},
		{
			name: "system message exists",
			input: []types.Message{
				{Role: types.RoleUser, Content: "user"},
				{Role: types.RoleSystem, Content: "system"},
			},
			expect: types.Message{Role: types.RoleSystem, Content: "system"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetSystemMsg(tt.input)
			if got.Role != tt.expect.Role || got.Content != tt.expect.Content {
				t.Errorf("got %v, expected %v", got, tt.expect)
			}
		})
	}
}

func TestTruncateContent(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		maxLen   int
		expected string
	}{
		{
			name:     "empty string",
			content:  "",
			maxLen:   10,
			expected: "",
		},
		{
			name:     "shorter than max",
			content:  "short",
			maxLen:   10,
			expected: "short",
		},
		{
			name:     "equal to max",
			content:  "1234567890",
			maxLen:   10,
			expected: "1234567890",
		},
		{
			name:     "longer than max",
			content:  "This is a very long string",
			maxLen:   10,
			expected: "This is a ...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TruncateContent(tt.content, tt.maxLen)
			if got != tt.expected {
				t.Errorf("got %q, expected %q", got, tt.expected)
			}
		})
	}
}

func TestGetLatestUserMsg(t *testing.T) {
	tests := []struct {
		name      string
		messages  []types.Message
		want      string
		expectErr bool
	}{
		{
			name:      "no messages",
			messages:  []types.Message{},
			expectErr: true,
		},
		{
			name: "only system message",
			messages: []types.Message{
				{Role: types.RoleSystem, Content: "system"},
			},
			expectErr: true,
		},
		{
			name: "one user message",
			messages: []types.Message{
				{Role: types.RoleSystem, Content: "system"},
				{Role: types.RoleUser, Content: "user"},
			},
			want:      "user",
			expectErr: false,
		},
		{
			name: "multiple user messages",
			messages: []types.Message{
				{Role: types.RoleSystem, Content: "system"},
				{Role: types.RoleUser, Content: "user1"},
				{Role: types.RoleUser, Content: "user2"},
			},
			want:      "user2",
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetLatestUserMsg(tt.messages)
			if (err != nil) != tt.expectErr {
				t.Errorf("GetLatestUserMsg() error = %v, wantErr %v", err, tt.expectErr)
				return
			}
			if !tt.expectErr && got != tt.want {
				t.Errorf("GetLatestUserMsg() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetOldUserMsgsWithNum(t *testing.T) {
	tests := []struct {
		name     string
		messages []types.Message
		num      int
		want     []types.Message
	}{
		{
			name:     "empty messages",
			messages: []types.Message{},
			num:      1,
			want:     []types.Message{},
		},
		{
			name: "no system message",
			messages: []types.Message{
				{Role: types.RoleUser, Content: "user1"},
			},
			num:  1,
			want: []types.Message{},
		},
		{
			name: "messages between system and user position",
			messages: []types.Message{
				{Role: types.RoleSystem, Content: "system"},
				{Role: types.RoleUser, Content: "user1"},
				{Role: types.RoleUser, Content: "user2"},
				{Role: types.RoleUser, Content: "user3"},
			},
			num: 1,
			want: []types.Message{
				{Role: types.RoleUser, Content: "user1"},
				{Role: types.RoleUser, Content: "user2"},
			},
		},
		{
			name: "invalid num",
			messages: []types.Message{
				{Role: types.RoleSystem, Content: "system"},
				{Role: types.RoleUser, Content: "user1"},
			},
			num:  0,
			want: []types.Message{{Role: types.RoleSystem, Content: "system"}, {Role: types.RoleUser, Content: "user1"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetOldUserMsgsWithNum(tt.messages, tt.num)
			if len(got) != len(tt.want) {
				t.Errorf("expected %d messages, got %d", len(tt.want), len(got))
				return
			}
			for i := range got {
				if got[i].Content != tt.want[i].Content || got[i].Role != tt.want[i].Role {
					t.Errorf("message %d mismatch: got %v, expected %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestGetRecentUserMsgsWithNum(t *testing.T) {
	tests := []struct {
		name     string
		messages []types.Message
		num      int
		want     []types.Message
	}{
		{
			name:     "empty messages",
			messages: []types.Message{},
			num:      1,
			want:     []types.Message{},
		},
		{
			name: "no user messages",
			messages: []types.Message{
				{Role: types.RoleSystem, Content: "system"},
			},
			num:  1,
			want: []types.Message{},
		},
		{
			name: "get single message",
			messages: []types.Message{
				{Role: types.RoleSystem, Content: "system"},
				{Role: types.RoleUser, Content: "user1"},
				{Role: types.RoleUser, Content: "user2"},
			},
			num: 1,
			want: []types.Message{
				{Role: types.RoleUser, Content: "user2"},
			},
		},
		{
			name: "get multiple messages",
			messages: []types.Message{
				{Role: types.RoleSystem, Content: "system"},
				{Role: types.RoleUser, Content: "user1"},
				{Role: types.RoleUser, Content: "user2"},
				{Role: types.RoleUser, Content: "user3"},
			},
			num: 2,
			want: []types.Message{
				{Role: types.RoleUser, Content: "user2"},
				{Role: types.RoleUser, Content: "user3"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetRecentUserMsgsWithNum(tt.messages, tt.num)
			if len(got) != len(tt.want) {
				t.Errorf("expected %d messages, got %d", len(tt.want), len(got))
				return
			}
			for i := range got {
				if got[i].Content != tt.want[i].Content || got[i].Role != tt.want[i].Role {
					t.Errorf("message %d mismatch: got %v, expected %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}
