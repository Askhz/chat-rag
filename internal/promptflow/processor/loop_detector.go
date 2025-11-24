package processor

import (
	"fmt"
	"strings"

	"github.com/zgsm-ai/chat-rag/internal/logger"
	"github.com/zgsm-ai/chat-rag/internal/model"
	"github.com/zgsm-ai/chat-rag/internal/types"
	"github.com/zgsm-ai/chat-rag/internal/utils"
	"go.uber.org/zap"
)

// LoopDetector is a processor that detects and handles loops in assistant responses
type LoopDetector struct {
	BaseProcessor
}

// NewLoopDetector creates a new LoopDetector processor
func NewLoopDetector() *LoopDetector {
	return &LoopDetector{}
}

// Execute processes the prompt message to detect and handle loops
func (l *LoopDetector) Execute(promptMsg *PromptMsg) {
	const method = "LoopDetector.Execute"

	if promptMsg == nil {
		l.Err = fmt.Errorf("received prompt message is empty")
		logger.Error(l.Err.Error(), zap.String("method", method))
		return
	}

	// Check for loops in assistant messages
	l.detectAndHandleLoops(promptMsg)

	l.Handled = true
	l.passToNext(promptMsg)
}

// detectAndHandleLoops checks if the last two assistant messages have the same content
// and adds a user message to break the loop if detected
func (l *LoopDetector) detectAndHandleLoops(promptMsg *PromptMsg) {
	const method = "LoopDetector.detectAndHandleLoops"

	// Skip processing if there are fewer than 3 messages
	if len(promptMsg.olderUserMsgList) < 3 {
		return
	}

	// Find the last two assistant messages
	var assistantMessages []types.Message
	for i := len(promptMsg.olderUserMsgList) - 1; i >= 0 && len(assistantMessages) < 2; i-- {
		msg := promptMsg.olderUserMsgList[i]
		if msg.Role == types.RoleAssistant {
			assistantMessages = append([]types.Message{msg}, assistantMessages...)
		}
	}

	// If we don't have two assistant messages, skip processing
	if len(assistantMessages) < 2 {
		return
	}

	// Extract content from the two assistant messages
	firstContent := utils.GetContentAsString(assistantMessages[0].Content)
	secondContent := utils.GetContentAsString(assistantMessages[1].Content)

	// Compare the content of the two assistant messages
	if strings.TrimSpace(firstContent) == strings.TrimSpace(secondContent) {
		logger.Info("Detected loop in assistant responses, adding intervention message",
			zap.String("method", method))

		// Add intervention content to the last user message
		err := l.addInterventionToUserMessage(promptMsg)
		if err != nil {
			logger.Error("Failed to add intervention to user message",
				zap.String("method", method),
				zap.Error(err))
		}
	}
}

// addInterventionToUserMessage adds intervention content to the user message
func (l *LoopDetector) addInterventionToUserMessage(promptMsg *PromptMsg) error {
	if promptMsg.lastUserMsg == nil {
		return fmt.Errorf("last user message is nil")
	}

	// Extract message contents
	var contentExtractor model.Content
	contents, err := contentExtractor.ExtractMsgContent(promptMsg.lastUserMsg)
	if err != nil {
		return fmt.Errorf("failed to extract message contents: %w", err)
	}

	// Add intervention content
	interventionContent := model.Content{
		Type: model.ContTypeText,
		Text: "Stop trying repetitive actions and rethink the actions to take. You can use different tools, and if you're unsure of the user's intent or goal, you can ask questions.",
	}

	contents = append(contents, interventionContent)
	promptMsg.lastUserMsg.Content = contents

	logger.Info("Added intervention content to user message")
	return nil
}
