package proxy

import (
	"bytes"
	"context"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashir500/Fuse/internal/config"
	"github.com/hashir500/Fuse/internal/store"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

type testProxy struct {
	server *Server
	store  *store.Store
}

func (p *testProxy) Close() {
	_ = p.store.Close()
}

func (p *testProxy) Do(req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	p.server.ServeHTTP(rec, req)
	return rec
}

func TestProxyForwardsRequest(t *testing.T) {
	t.Parallel()

	proxy := startTestProxy(t, func(r *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, `{"ok": true, "usage": {"input_tokens": 1, "output_tokens": 1}}`, r), nil
	})
	defer proxy.Close()

	req := httptest.NewRequest(http.MethodPost, "http://localhost:8787/v1/messages", strings.NewReader(`{"model":"claude-test","max_tokens":10,"messages":[{"role":"user","content":"hi"}]}`))
	req.Header.Set("Content-Type", "application/json")
	rec := proxy.Do(req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"ok": true`) {
		t.Fatalf("expected upstream body, got %s", rec.Body.String())
	}
}

func TestProxyPreservesAuthHeaders(t *testing.T) {
	t.Parallel()

	var receivedHeaders http.Header
	proxy := startTestProxy(t, func(r *http.Request) (*http.Response, error) {
		receivedHeaders = r.Header.Clone()
		return jsonResponse(http.StatusOK, `{"usage": {"input_tokens": 1, "output_tokens": 1}}`, r), nil
	})
	defer proxy.Close()

	req := httptest.NewRequest(http.MethodPost, "http://localhost:8787/v1/messages", strings.NewReader(`{"model":"claude-test","max_tokens":10,"messages":[{"role":"user","content":"hi"}]}`))
	req.Header.Set("x-api-key", "sk-test-secret-key")
	req.Header.Set("authorization", "Bearer test-token")
	req.Header.Set("content-type", "application/json")
	proxy.Do(req)

	if receivedHeaders.Get("x-api-key") != "sk-test-secret-key" {
		t.Errorf("x-api-key not forwarded: got %q", receivedHeaders.Get("x-api-key"))
	}
	if receivedHeaders.Get("authorization") != "Bearer test-token" {
		t.Errorf("authorization not forwarded: got %q", receivedHeaders.Get("authorization"))
	}
}

func TestProxyHardCapBlocksBeforeUpstream(t *testing.T) {
	t.Parallel()

	upstreamHit := false
	cfg := anthropicTestConfig()
	cfg.Budgets.Daily = config.Budget{Hard: 0.001}
	proxy := startTestProxyWithConfig(t, cfg, func(r *http.Request) (*http.Response, error) {
		upstreamHit = true
		return jsonResponse(http.StatusOK, `{"ok": true}`, r), nil
	}, io.Discard)
	defer proxy.Close()

	body := `{"model":"claude-test","max_tokens":1000,"messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest(http.MethodPost, "http://localhost:8787/v1/messages", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := proxy.Do(req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rec.Code)
	}
	if upstreamHit {
		t.Fatal("upstream was hit; hard cap did not block before forwarding")
	}
	if !strings.Contains(rec.Body.String(), "fuse_hard_cap_exceeded") {
		t.Fatalf("expected fuse error, got: %s", rec.Body.String())
	}
}

func TestProxyLogsActualCost(t *testing.T) {
	t.Parallel()

	proxy := startTestProxy(t, func(r *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, `{"usage": {"input_tokens": 10, "output_tokens": 20}}`, r), nil
	})
	defer proxy.Close()

	body := `{"model":"claude-test","max_tokens":100,"messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest(http.MethodPost, "http://localhost:8787/v1/messages", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := proxy.Do(req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	reqLog := getLastRequestFromDB(t, proxy.store)
	expectedCost := 0.00033
	if math.Abs(reqLog.EstimatedCost-expectedCost) > 0.00001 {
		t.Fatalf("expected cost ~%.5f, got %.5f", expectedCost, reqLog.EstimatedCost)
	}
	if reqLog.WasBlocked {
		t.Fatal("request should not be blocked")
	}
}

func TestProxySoftCapWarnsButAllows(t *testing.T) {
	t.Parallel()

	cfg := anthropicTestConfig()
	cfg.Budgets.Daily = config.Budget{Soft: 0.0001, Hard: 1.00}
	var stderr bytes.Buffer
	proxy := startTestProxyWithConfig(t, cfg, func(r *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, `{"usage": {"input_tokens": 1, "output_tokens": 1}}`, r), nil
	}, &stderr)
	defer proxy.Close()

	if err := proxy.store.LogRequest(context.Background(), store.RequestLog{
		Provider:      "anthropic",
		Model:         "claude-test",
		EstimatedCost: 0.00009,
	}); err != nil {
		t.Fatal(err)
	}

	body := `{"model":"claude-test","max_tokens":10,"messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest(http.MethodPost, "http://localhost:8787/v1/messages", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := proxy.Do(req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(stderr.String(), "SOFT CAP") {
		t.Fatalf("expected soft cap warning, got: %s", stderr.String())
	}
}

func TestProxyGeminiUsageParsing(t *testing.T) {
	t.Parallel()

	cfg := googleTestConfig()
	proxy := startTestProxyWithConfig(t, cfg, func(r *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, `{
			"modelVersion":"gemini-test",
			"candidates":[{"content":{"parts":[{"text":"Hello"}]}}],
			"usageMetadata":{"promptTokenCount":7,"candidatesTokenCount":1,"thoughtsTokenCount":25,"totalTokenCount":33}
		}`, r), nil
	}, io.Discard)
	defer proxy.Close()

	body := `{"contents":[{"parts":[{"text":"Say hello"}]}],"generationConfig":{"maxOutputTokens":30}}`
	req := httptest.NewRequest(http.MethodPost, "http://localhost:8787/v1beta/models/gemini-test:generateContent", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := proxy.Do(req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	reqLog := getLastRequestFromDB(t, proxy.store)
	expectedCost := 0.0000107
	if math.Abs(reqLog.EstimatedCost-expectedCost) > 0.000001 {
		t.Fatalf("expected Gemini cost ~%.7f, got %.7f", expectedCost, reqLog.EstimatedCost)
	}
	if reqLog.PromptTokens != 7 {
		t.Fatalf("expected prompt tokens 7, got %d", reqLog.PromptTokens)
	}
	if reqLog.CompletionTokens != 26 {
		t.Fatalf("expected completion tokens 26, got %d", reqLog.CompletionTokens)
	}
}

func startTestProxy(t *testing.T, roundTrip roundTripFunc) *testProxy {
	t.Helper()
	return startTestProxyWithConfig(t, anthropicTestConfig(), roundTrip, io.Discard)
}

func startTestProxyWithConfig(t *testing.T, cfg *config.Config, roundTrip roundTripFunc, stderr io.Writer) *testProxy {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "spend.db"))
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{
		Config: cfg,
		Store:  db,
		Stderr: stderr,
		Client: &http.Client{Transport: roundTrip},
	}
	return &testProxy{server: server, store: db}
}

func getLastRequestFromDB(t *testing.T, db *store.Store) store.RequestLog {
	t.Helper()
	logs, err := db.Recent(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 request, got %d", len(logs))
	}
	return logs[0]
}

func jsonResponse(status int, body string, req *http.Request) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}
}

func anthropicTestConfig() *config.Config {
	return &config.Config{
		Providers: map[string]config.ProviderConfig{
			"anthropic": {
				BaseURL: "https://api.anthropic.test",
				Models: map[string]config.ModelCosts{
					"claude-test": {
						InputCostPer1K:  0.003,
						OutputCostPer1K: 0.015,
					},
				},
			},
		},
		Budgets: config.BudgetConfig{
			Daily:   config.Budget{Hard: 50},
			Weekly:  config.Budget{Hard: 200},
			Monthly: config.Budget{Hard: 500},
		},
		Estimation: config.EstimationConfig{Mode: "max", OutputRatio: 0.3},
		OnHardCap:  "block",
	}
}

func googleTestConfig() *config.Config {
	return &config.Config{
		Providers: map[string]config.ProviderConfig{
			"google": {
				BaseURL: "https://generativelanguage.test",
				Models: map[string]config.ModelCosts{
					"gemini-test": {
						InputCostPer1K:  0.0001,
						OutputCostPer1K: 0.0004,
					},
				},
			},
		},
		Budgets: config.BudgetConfig{
			Daily:   config.Budget{Hard: 50},
			Weekly:  config.Budget{Hard: 200},
			Monthly: config.Budget{Hard: 500},
		},
		Estimation: config.EstimationConfig{Mode: "max", OutputRatio: 0.3},
		OnHardCap:  "block",
	}
}
