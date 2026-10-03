package apicompat

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResponsesToAnthropicRequest_Opus55UsesAdaptiveThinking(t *testing.T) {
	req := &ResponsesRequest{
		Model:     "claude-opus-5-5",
		Input:     json.RawMessage(`[{"role":"user","content":"Hello"}]`),
		Reasoning: &ResponsesReasoning{Effort: "high"},
	}

	out, err := ResponsesToAnthropicRequest(req)
	require.NoError(t, err)
	require.NotNil(t, out.Thinking)
	assert.Equal(t, "adaptive", out.Thinking.Type)
	require.NotNil(t, out.OutputConfig)
	assert.Equal(t, "high", out.OutputConfig.Effort)
}

func TestResponsesToAnthropicRequest_Opus55ReplaysLocalThinkingEnvelope(t *testing.T) {
	envelope := encodeAnthropicThinking(AnthropicContentBlock{
		Type: "thinking", Thinking: "plan", Signature: "signed",
	})
	req := &ResponsesRequest{
		Model: "claude-opus-5-5",
		Input: json.RawMessage(`[{"type":"reasoning","encrypted_content":"` + envelope + `"}]`),
	}

	out, err := ResponsesToAnthropicRequest(req)
	require.NoError(t, err)
	require.Len(t, out.Messages, 1)
	assert.Equal(t, "assistant", out.Messages[0].Role)
	assert.Contains(t, string(out.Messages[0].Content), `"signature":"signed"`)
}

func TestResponsesToAnthropicRequest_RejectsExternalThinkingCiphertext(t *testing.T) {
	req := &ResponsesRequest{
		Model: "claude-opus-5-5",
		Input: json.RawMessage(`[{"type":"reasoning","encrypted_content":"external-ciphertext"}]`),
	}

	out, err := ResponsesToAnthropicRequest(req)
	require.NoError(t, err)
	assert.Empty(t, out.Messages)
}

func TestResponsesToAnthropicRequest_Opus55RejectsForcedTool(t *testing.T) {
	req := &ResponsesRequest{
		Model:      "claude-opus-5-5",
		Input:      json.RawMessage(`[{"role":"user","content":"Use a tool"}]`),
		ToolChoice: json.RawMessage(`"required"`),
	}

	_, err := ResponsesToAnthropicRequest(req)
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "forced tool_choice"))
}
