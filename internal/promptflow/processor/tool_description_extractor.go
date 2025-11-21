package processor

import (
	"strings"

	"github.com/zgsm-ai/chat-rag/internal/logger"
	"github.com/zgsm-ai/chat-rag/internal/model"
	"github.com/zgsm-ai/chat-rag/internal/types"
	"go.uber.org/zap"
)

const (
	// ToolDescStartPattern defines the start pattern for tool description
	ToolDescStartPattern = "====\n\nTOOL USE"
	// ToolDescEndPattern defines the end pattern for tool description
	ToolDescEndPattern = "\n\n====\n\nCAPABILITIES"
	// ApplyDiffErrorMessage defines the error message for apply_diff parsing failure
	ApplyDiffErrorMessage = "Failed to parse apply_diff XML"
	// ApplyDiffSectionPattern defines the start pattern for apply_diff section
	ApplyDiffSectionPattern = "## apply_diff"
	// ApplyDiffNextSectionPattern defines the pattern for the next section
	ApplyDiffNextSectionPattern = "\n## "
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

	systemContent, err := t.extractSystemContent(promptMsg.GetSystemMsg())
	if err != nil {
		logger.Error("Failed to extract system content", zap.Error(err))
		t.passToNext(promptMsg)
		return
	}

	startIndex, endIndex := t.findToolDescriptionBounds(systemContent)
	if startIndex == -1 || endIndex == -1 {
		logger.Info("No tool description found in system message")
		t.passToNext(promptMsg)
		return
	}

	toolDescription := systemContent[startIndex:endIndex]

	newSystemContent := systemContent[:startIndex] + systemContent[endIndex:]

	err = t.addToolDescriptionToUserMessage(promptMsg, toolDescription)
	if err != nil {
		logger.Error("Failed to add tool description to user message", zap.Error(err))
		t.passToNext(promptMsg)
		return
	}

	promptMsg.UpdateSystemMsg(newSystemContent)
	t.passToNext(promptMsg)
}

// addToolDescriptionToUserMessage adds appropriate content to the user message
// based on whether apply_diff error is detected
func (t *ToolDescriptionExtractor) addToolDescriptionToUserMessage(promptMsg *PromptMsg, toolDescription string) error {
	if promptMsg.lastUserMsg == nil {
		return nil
	}

	contents, err := t.extractMessageContents(promptMsg.lastUserMsg)
	if err != nil {
		return err
	}

	if t.hasApplyDiffError(contents) {
		contents = t.addApplyDiffUsage(contents, toolDescription)
	} else {
		contents = t.addToolDescription(contents, toolDescription)
	}

	promptMsg.lastUserMsg.Content = contents
	return nil
}

// extractMessageContents extracts and normalizes message contents
func (t *ToolDescriptionExtractor) extractMessageContents(message *types.Message) ([]model.Content, error) {
	var contentExtractor model.Content
	return contentExtractor.ExtractMsgContent(message)
}

// hasApplyDiffError checks if any content contains apply_diff parsing error
func (t *ToolDescriptionExtractor) hasApplyDiffError(contents []model.Content) bool {
	for _, content := range contents {
		if strings.Contains(content.Text, ApplyDiffErrorMessage) {
			return true
		}
	}
	return false
}

// addApplyDiffUsage extracts and adds apply_diff usage to contents
func (t *ToolDescriptionExtractor) addApplyDiffUsage(contents []model.Content, toolDescription string) []model.Content {
	logger.Info("Detected apply_diff parsing error in user message, adding apply_diff usage")

	applyDiffContent := t.extractApplyDiffSection(toolDescription)
	if applyDiffContent == "" {
		logger.Warn("Could not extract apply_diff section from tool description")
		return contents
	}

	applyDiffUsageContent := model.Content{
		Type:         model.ContTypeText,
		Text:         "<apply_diff_usage>\n" + applyDiffContent + "</apply_diff_usage>",
		CacheControl: model.EphemeralCacheControl,
	}

	contents = append(contents, applyDiffUsageContent)
	logger.Info("Added apply_diff usage to user message due to parse error")
	return contents
}

// extractApplyDiffSection extracts the apply_diff section from tool description
func (t *ToolDescriptionExtractor) extractApplyDiffSection(toolDescription string) string {
	startIndex := strings.Index(toolDescription, ApplyDiffSectionPattern)
	if startIndex == -1 {
		return ""
	}

	endIndex := strings.Index(toolDescription[startIndex:], ApplyDiffNextSectionPattern)
	if endIndex == -1 {
		return ""
	}

	endIndex += startIndex
	return toolDescription[startIndex:endIndex]
}

// addToolDescription adds the complete tool description to contents
func (t *ToolDescriptionExtractor) addToolDescription(contents []model.Content, toolDescription string) []model.Content {
	toolDescContent := model.Content{
		Type:         model.ContTypeText,
		Text:         "<tool_description>\n" + toolDescription + "</tool_description>",
		CacheControl: model.EphemeralCacheControl,
	}

	contents = append(contents, toolDescContent)
	logger.Info("Added tool_description content to user message", zap.Int("length", len(toolDescription)))
	return contents
}

// findToolDescriptionBounds finds the start and end positions of tool description in system content
// Returns start index and end index (exclusive), or -1, -1 if not found
func (t *ToolDescriptionExtractor) findToolDescriptionBounds(systemContent string) (int, int) {
	startIndex := strings.Index(systemContent, ToolDescStartPattern)
	if startIndex == -1 {
		return -1, -1
	}

	endIndex := strings.Index(systemContent[startIndex:], ToolDescEndPattern)
	if endIndex == -1 {
		return -1, -1
	}

	endIndex += startIndex
	return startIndex, endIndex
}
