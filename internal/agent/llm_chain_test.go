package agent

import (
	"context"
	"errors"
	"testing"
)

func TestLLMProviderConstants(t *testing.T) {
	providers := []LLMProvider{
		ProviderGPT5, ProviderClaudeSonnet, ProviderGemini, ProviderStatic,
	}
	if len(providers) != 4 {
		t.Errorf("LLMProvider count = %d, want 4", len(providers))
	}
}

func TestDegradationChainSuccess(t *testing.T) {
	chain := NewDegradationChain(nil, "fallback")

	gpt5 := NewMockCaller(ProviderGPT5, &LLMResponse{
		Content:  "gpt5 response",
		Provider: ProviderGPT5,
	}, nil)
	chain.AddCaller(gpt5)

	resp, err := chain.Call(context.Background(), &LLMRequest{Prompt: "test"})
	if err != nil {
		t.Fatalf("Call should succeed: %v", err)
	}
	if resp.Content != "gpt5 response" {
		t.Errorf("Content = %s, want gpt5 response", resp.Content)
	}
	if resp.Provider != ProviderGPT5 {
		t.Errorf("Provider = %s, want %s", resp.Provider, ProviderGPT5)
	}
	if gpt5.CallCount() != 1 {
		t.Errorf("GPT5 callCount = %d, want 1", gpt5.CallCount())
	}
}

func TestDegradationChainFallback(t *testing.T) {
	chain := NewDegradationChain(nil, "static fallback")

	gpt5 := NewMockCaller(ProviderGPT5, nil, errors.New("gpt5 down"))
	claude := NewMockCaller(ProviderClaudeSonnet, &LLMResponse{
		Content:  "claude response",
		Provider: ProviderClaudeSonnet,
	}, nil)
	chain.AddCaller(gpt5)
	chain.AddCaller(claude)

	resp, err := chain.Call(context.Background(), &LLMRequest{Prompt: "test"})
	if err != nil {
		t.Fatalf("Call should succeed: %v", err)
	}
	if resp.Content != "claude response" {
		t.Errorf("Content = %s, want claude response", resp.Content)
	}
	if gpt5.CallCount() != 1 {
		t.Errorf("GPT5 callCount = %d, want 1", gpt5.CallCount())
	}
	if claude.CallCount() != 1 {
		t.Errorf("Claude callCount = %d, want 1", claude.CallCount())
	}
}

func TestDegradationChainAllFailUseStatic(t *testing.T) {
	chain := NewDegradationChain(nil, "static template content")

	gpt5 := NewMockCaller(ProviderGPT5, nil, errors.New("gpt5 down"))
	claude := NewMockCaller(ProviderClaudeSonnet, nil, errors.New("claude down"))
	gemini := NewMockCaller(ProviderGemini, nil, errors.New("gemini down"))
	chain.AddCaller(gpt5)
	chain.AddCaller(claude)
	chain.AddCaller(gemini)

	resp, err := chain.Call(context.Background(), &LLMRequest{Prompt: "test"})
	if err != nil {
		t.Fatalf("Call should return static fallback: %v", err)
	}
	if resp.Content != "static template content" {
		t.Errorf("Content = %s, want static template content", resp.Content)
	}
	if resp.Provider != ProviderStatic {
		t.Errorf("Provider = %s, want %s", resp.Provider, ProviderStatic)
	}
	if !resp.IsFallback {
		t.Error("IsFallback should be true")
	}
	if gpt5.CallCount() != 1 {
		t.Errorf("GPT5 callCount = %d, want 1", gpt5.CallCount())
	}
	if claude.CallCount() != 1 {
		t.Errorf("Claude callCount = %d, want 1", claude.CallCount())
	}
	if gemini.CallCount() != 1 {
		t.Errorf("Gemini callCount = %d, want 1", gemini.CallCount())
	}
}

func TestDegradationChainPriority(t *testing.T) {
	chain := NewDegradationChain(nil, "fallback")

	// Add in reverse priority to verify order matters
	gemini := NewMockCaller(ProviderGemini, &LLMResponse{
		Content:  "gemini response",
		Provider: ProviderGemini,
	}, nil)
	claude := NewMockCaller(ProviderClaudeSonnet, &LLMResponse{
		Content:  "claude response",
		Provider: ProviderClaudeSonnet,
	}, nil)
	chain.AddCaller(claude)
	chain.AddCaller(gemini)

	resp, err := chain.Call(context.Background(), &LLMRequest{Prompt: "test"})
	if err != nil {
		t.Fatalf("Call should succeed: %v", err)
	}
	// Claude should be called first (added first)
	if resp.Content != "claude response" {
		t.Errorf("Content = %s, want claude response", resp.Content)
	}
	if claude.CallCount() != 1 {
		t.Errorf("Claude callCount = %d, want 1", claude.CallCount())
	}
	if gemini.CallCount() != 0 {
		t.Errorf("Gemini callCount = %d, want 0", gemini.CallCount())
	}
}

func TestMockCallerProvider(t *testing.T) {
	caller := NewMockCaller(ProviderGPT5, nil, nil)
	if caller.Provider() != ProviderGPT5 {
		t.Errorf("Provider = %s, want %s", caller.Provider(), ProviderGPT5)
	}
}

func TestMockCallerCallCount(t *testing.T) {
	caller := NewMockCaller(ProviderGPT5, &LLMResponse{Content: "ok"}, nil)
	if caller.CallCount() != 0 {
		t.Errorf("Initial callCount = %d, want 0", caller.CallCount())
	}

	_, _ = caller.Call(context.Background(), &LLMRequest{Prompt: "test"})
	if caller.CallCount() != 1 {
		t.Errorf("After 1 call, callCount = %d, want 1", caller.CallCount())
	}
}
