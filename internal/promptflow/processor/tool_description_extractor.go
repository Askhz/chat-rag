package processor

import (
	"strings"

	"github.com/zgsm-ai/chat-rag/internal/logger"
	"github.com/zgsm-ai/chat-rag/internal/model"
	"go.uber.org/zap"
)

const (
	// ToolDescStartPattern defines the start pattern for tool description
	ToolDescStartPattern = "====\n\nTOOL USE"
	// ToolDescEndPattern defines the end pattern for tool description
	ToolDescEndPattern = "\n\n====\n\nCAPABILITIES"
)

// ToolDescriptionExtractor extracts tool descriptions from system message
// and adds them to the user message
type ToolDescriptionExtractor struct {
	BaseProcessor
}

// NewToolDescriptionExtractor creates a new ToolDescriptionExtractor processor
func NewToolDescriptionExtractor() *ToolDescriptionExtractor {
	return &ToolDescriptionExtractor{}
}

// Execute processes the prompt message to extract tool descriptions
func (t *ToolDescriptionExtractor) Execute(promptMsg *PromptMsg) {
	logger.Info("Executing ToolDescriptionExtractor")

	// Extract system content
	systemContent, err := t.extractSystemContent(promptMsg.GetSystemMsg())
	if err != nil {
		logger.Error("Failed to extract system content", zap.Error(err))
		t.passToNext(promptMsg)
		return
	}

	// Find tool description bounds in system content
	startIndex, endIndex := t.findToolDescriptionBounds(systemContent)
	if startIndex == -1 || endIndex == -1 {
		logger.Info("No tool description found in system message")
		t.passToNext(promptMsg)
		return
	}

	// Extract the tool description
	toolDescription := systemContent[startIndex:endIndex]
	logger.Info("Tool description extracted",
		zap.Int("length", len(toolDescription)))

	// Remove tool description from system message
	newSystemContent := systemContent[:startIndex] + systemContent[endIndex:]

	// Add tool description to user message
	err = t.addToolDescriptionToUserMessage(promptMsg, toolDescription)
	if err != nil {
		logger.Error("Failed to add tool description to user message", zap.Error(err))
		t.passToNext(promptMsg)
		return
	}

	// Update system message only after successfully adding to user message
	// This ensures atomicity - either both operations succeed or neither does
	promptMsg.UpdateSystemMsg(newSystemContent)

	t.passToNext(promptMsg)
}

// addToolDescriptionToUserMessage adds tool description to the user message
func (t *ToolDescriptionExtractor) addToolDescriptionToUserMessage(promptMsg *PromptMsg, toolDescription string) error {
	// Get the last user message
	if promptMsg.lastUserMsg == nil {
		return nil // No user message to modify
	}

	// Use ExtractMsgContent to normalize content to []Content
	var contentExtractor model.Content
	contents, err := contentExtractor.ExtractMsgContent(promptMsg.lastUserMsg)
	if err != nil {
		return err
	}

	// Create tool description content with tags
	toolDescContent := model.Content{
		Type:         model.ContTypeText,
		Text:         "<tool_description>\n" + toolDescription + "</tool_description>",
		CacheControl: model.EphemeralCacheControl,
	}

	// Add tool description to the end of the content list
	contents = append(contents, toolDescContent)

	// Update the user message content
	promptMsg.lastUserMsg.Content = contents

	logger.Info("Added tool description to user message",
		zap.Int("content_count", len(contents)))

	return nil
}

// findToolDescriptionBounds finds the start and end positions of tool description in system content
// Returns start index and end index (exclusive), or -1, -1 if not found
func (t *ToolDescriptionExtractor) findToolDescriptionBounds(systemContent string) (int, int) {
	// Find the start position
	startIndex := strings.Index(systemContent, ToolDescStartPattern)
	if startIndex == -1 {
		return -1, -1
	}

	// Find the end position after the start position (before ToolDescEndPattern)
	endIndex := strings.Index(systemContent[startIndex:], ToolDescEndPattern)
	if endIndex == -1 {
		return -1, -1
	}

	// Calculate the actual end position in the original string (before ToolDescEndPattern)
	endIndex += startIndex

	return startIndex, endIndex
}
