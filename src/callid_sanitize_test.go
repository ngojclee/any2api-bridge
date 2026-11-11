package main

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

func TestNormalizeOpenAIRequestCallIDsClampsResponsesAndStripsSyntheticItemID(t *testing.T) {
	cpaTID := "cpa_tid_v1_ZXhlY19jb21tYW5kOjAjMzQ1OWIwMjBiOThlNDIwMTkwY2E0MTAxOTk0NzYzMDY"
	if len(cpaTID) != 74 {
		t.Fatalf("expected 74-char fixture, got %d", len(cpaTID))
	}
	rawBody := []byte(`{
		"model":"gpt-6.1-terra",
		"input":[
			{"type":"message","id":"msg_valid123","role":"user","content":[{"type":"input_text","text":"hi"}]},
			{"type":"function_call","id":"` + cpaTID + `","call_id":"` + cpaTID + `","name":"exec_command","arguments":"{}"},
			{"type":"function_call_output","call_id":"` + cpaTID + `","output":"ok"}
		]
	}`)

	normalized, changed := normalizeOpenAIRequestCallIDs(rawBody)
	if !changed {
		t.Fatal("expected oversized CPA call_id/id to be normalized")
	}
	var parsed map[string]any
	if err := json.Unmarshal(normalized, &parsed); err != nil {
		t.Fatalf("unmarshal normalized: %v", err)
	}
	items := parsed["input"].([]any)
	msgItem := items[0].(map[string]any)
	if msgItem["id"] != "msg_valid123" {
		t.Fatalf("valid OpenAI message id was altered: %v", msgItem["id"])
	}
	callItem := items[1].(map[string]any)
	outItem := items[2].(map[string]any)
	if _, hasID := callItem["id"]; hasID {
		t.Fatalf("expected synthetic cpa_tid_v1_ id on function_call to be stripped, got %v", callItem["id"])
	}
	gotCallID, _ := callItem["call_id"].(string)
	gotOutCallID, _ := outItem["call_id"].(string)
	if gotCallID == "" || len(gotCallID) != 53 || !strings.HasPrefix(gotCallID, "call_") {
		t.Fatalf("unexpected clamped call_id %q (len=%d)", gotCallID, len(gotCallID))
	}
	if gotCallID != gotOutCallID {
		t.Fatalf("call/output call_id mismatch: %q vs %q", gotCallID, gotOutCallID)
	}

	// Second pass must be a fast no-op (idempotent).
	if _, secondChanged := normalizeOpenAIRequestCallIDs(normalized); secondChanged {
		t.Fatal("expected already-normalized payload to be a no-op")
	}
}

func TestHandleInterceptAfterClampsCodexProviderOversizedCallIDsWithoutTouchingDevin(t *testing.T) {
	withSettings(t, PluginSettings{
		Enabled:           true,
		DirectModeEnabled: true,
	})
	cpaTID := "cpa_tid_v1_ZXhlY19jb21tYW5kOjAjMzQ1OWIwMjBiOThlNDIwMTkwY2E0MTAxOTk0NzYzMDY"
	body := []byte(`{"model":"gpt-6.1-terra","input":[{"type":"function_call","id":"` + cpaTID + `","call_id":"` + cpaTID + `","name":"exec_command","arguments":"{}"}]}`)

	// 1. Codex target -> rewrites Body even though Codex OAuth is not a direct_account.
	reqCodex, _ := json.Marshal(map[string]any{
		"source_format":   "openai-response",
		"to_format":       "codex",
		"model":           "gpt-6.1-terra",
		"requested_model": "gpt-6.1-terra",
		"body":            base64.StdEncoding.EncodeToString(body),
	})
	respRaw, err := handleInterceptAfter(reqCodex)
	if err != nil {
		t.Fatalf("handleInterceptAfter codex: %v", err)
	}
	var env struct {
		OK     bool `json:"ok"`
		Result struct {
			Body string `json:"Body"`
		} `json:"result"`
	}
	if err := json.Unmarshal(respRaw, &env); err != nil || !env.OK || env.Result.Body == "" {
		t.Fatalf("expected rewritten Body in codex response: %s", respRaw)
	}
	decodedBody, err := base64.StdEncoding.DecodeString(env.Result.Body)
	if err != nil {
		t.Fatalf("decode rewritten body: %v", err)
	}
	if strings.Contains(string(decodedBody), "cpa_tid_") {
		t.Fatalf("cpa_tid_ survived codex normalization: %s", decodedBody)
	}

	// 2. Devin target -> preserves cpa_tid_v1_* untouched (Body empty = keep original).
	reqDevin, _ := json.Marshal(map[string]any{
		"source_format":   "openai-response",
		"to_format":       "devin",
		"model":           "swe-2",
		"requested_model": "swe-2",
		"body":            base64.StdEncoding.EncodeToString(body),
	})
	respDevinRaw, err := handleInterceptAfter(reqDevin)
	if err != nil {
		t.Fatalf("handleInterceptAfter devin: %v", err)
	}
	env.Result.Body = ""
	if err := json.Unmarshal(respDevinRaw, &env); err != nil || !env.OK {
		t.Fatalf("unmarshal devin response: %v", err)
	}
	if env.Result.Body != "" {
		t.Fatalf("expected devin body to remain untouched, got %q", env.Result.Body)
	}
}
