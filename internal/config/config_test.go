package config

import (
	"strings"
	"testing"
)

func TestConfigValidatesRequiredFields(t *testing.T) {
	t.Parallel()

	cfg := Config{
		Providers: map[string]ProviderConfig{
			"anthropic": {
				BaseURL: "https://api.anthropic.com",
				Models: map[string]ModelCosts{
					"claude-test": {InputCostPer1K: 0.003, OutputCostPer1K: 0.015},
				},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error for missing api_key")
	}
	if !strings.Contains(err.Error(), "api_key") {
		t.Fatalf("expected error mentioning api_key, got: %v", err)
	}
}

func TestConfigValidatesModelRates(t *testing.T) {
	t.Parallel()

	cfg := Config{
		Providers: map[string]ProviderConfig{
			"anthropic": {
				BaseURL: "https://api.anthropic.com",
				APIKey:  "test-key",
				Models: map[string]ModelCosts{
					"claude-test": {InputCostPer1K: 0, OutputCostPer1K: 0},
				},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error for zero rates")
	}
	if !strings.Contains(err.Error(), "costs") {
		t.Fatalf("expected error mentioning costs, got: %v", err)
	}
}
