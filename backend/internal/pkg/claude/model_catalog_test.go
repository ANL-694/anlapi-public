package claude

import "testing"

func TestDefaultModelsContainsOpus55(t *testing.T) {
	for _, model := range DefaultModels {
		if model.ID == "claude-opus-5-5" {
			if model.DisplayName != "Claude Opus 5.5" || model.CreatedAt != "2026-09-22T00:00:00Z" {
				t.Fatalf("unexpected Opus 5.5 descriptor: %+v", model)
			}
			return
		}
	}
	t.Fatal("claude-opus-5-5 missing")
}

func TestDefaultModelsContainsSonnet55(t *testing.T) {
	for _, model := range DefaultModels {
		if model.ID == "claude-sonnet-5-5" {
			if model.DisplayName != "Claude Sonnet 5.5" || model.CreatedAt != "2026-09-28T00:00:00Z" {
				t.Fatalf("unexpected Sonnet 5.5 descriptor: %+v", model)
			}
			return
		}
	}
	t.Fatal("claude-sonnet-5-5 missing")
}

func TestIsOpus55(t *testing.T) {
	if !IsOpus55("models/claude-opus-5-5") {
		t.Fatal("models/ prefix should be accepted")
	}
	if IsOpus55("claude-opus-5-5-20260922") {
		t.Fatal("dated variants are not the fixed model ID")
	}
}
