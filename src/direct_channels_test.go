package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDirectModeInjectsDynamicHeaderFamiliesByAccountKind(t *testing.T) {
	previous := currentPluginSettings()
	t.Cleanup(func() {
		pluginSettingsMu.Lock()
		pluginSettings = previous
		pluginSettingsMu.Unlock()
	})
	settings := defaultPluginSettings()
	settings.DirectModeEnabled = true
	settings.HMACSecretSource = "config"
	settings.HMACSecret = "direct-signing-secret"
	settings.DirectAccounts = []directAccount{
		{
			AccountID:              "agy-prod",
			ProviderKind:           directProviderAGY,
			ChannelName:            "AGY Direct",
			Prefix:                 "agy",
			BaseURL:                "https://agy2api.example/v1",
			Enabled:                true,
			IdentitySigningEnabled: true,
			Models:                 []directAccountModel{{UpstreamID: "gemini-3.8-flash", Enabled: true}},
		},
		{
			AccountID:              "gpt-prod",
			ProviderKind:           directProviderGPT,
			ChannelName:            "GPT Direct",
			Prefix:                 "gpt",
			BaseURL:                "https://gpt2api.example/v1",
			Enabled:                true,
			IdentitySigningEnabled: true,
			Models:                 []directAccountModel{{UpstreamID: "gpt-5.6", Enabled: true}},
		},
	}
	pluginSettingsMu.Lock()
	pluginSettings = normalizeSettings(settings)
	pluginSettingsMu.Unlock()

	agy := interceptHeadersForTest(t, []byte(`{"ToFormat":"openai","RequestedModel":"agy/gemini-3.8-flash","Model":"gemini-3.8-flash","Headers":{"Authorization":["Bearer client-key"],"X-AGY-Client-App":["codex"]},"Metadata":{"request_path":"/v1/chat/completions"}}`))
	if firstHeaderValue(agy, "X-AGY-Principal") == "" || firstHeaderValue(agy, "X-AGY-Signature") == "" {
		t.Fatalf("AGY direct account missing AGY headers: %+v", agy)
	}
	if firstHeaderValue(agy, any2APIHeaderPrincipal) != "" || firstHeaderValue(agy, any2APIHeaderSignature) != "" {
		t.Fatalf("AGY direct account leaked Any2API headers: %+v", agy)
	}

	gpt := interceptHeadersForTest(t, []byte(`{"ToFormat":"openai","RequestedModel":"gpt/gpt-5.6","Model":"gpt-5.6","Headers":{"Authorization":["Bearer client-key"],"X-Any2API-Client-App":["codex"],"X-Any2API-Conversation-Id":["thread-a"]},"Metadata":{"request_path":"/v1/chat/completions"}}`))
	if firstHeaderValue(gpt, any2APIHeaderPrincipal) == "" ||
		firstHeaderValue(gpt, any2APIHeaderSignature) == "" ||
		firstHeaderValue(gpt, any2APIHeaderConversationID) != "thread-a" {
		t.Fatalf("GPT direct account missing Any2API headers: %+v", gpt)
	}
	for key := range gpt {
		if strings.HasPrefix(strings.ToLower(key), "x-agy-") {
			t.Fatalf("GPT direct account leaked AGY header %s: %+v", key, gpt)
		}
	}
}

func interceptHeadersForTest(t *testing.T, raw []byte) map[string][]string {
	t.Helper()
	response, errHandle := handleInterceptAfter(raw)
	if errHandle != nil {
		t.Fatal(errHandle)
	}
	var decoded struct {
		OK     bool `json:"ok"`
		Result struct {
			Headers map[string][]string `json:"Headers"`
		} `json:"result"`
	}
	if errUnmarshal := json.Unmarshal(response, &decoded); errUnmarshal != nil {
		t.Fatal(errUnmarshal)
	}
	if !decoded.OK {
		t.Fatalf("intercept failed: %s", response)
	}
	return decoded.Result.Headers
}

func TestDirectModeDoesNotAdvertiseExecutorOrModelProvider(t *testing.T) {
	settings := defaultPluginSettings()
	settings.DirectModeEnabled = true
	settings.ExecutorEnabled = true
	settings.DirectAccounts = []directAccount{{
		AccountID:              "gpt-prod",
		ProviderKind:           directProviderGPT,
		ChannelName:            "GPT Direct",
		Prefix:                 "gpt",
		Enabled:                true,
		IdentitySigningEnabled: true,
	}}
	withSettings(t, settings)
	storeProviderSpec(providerSpec{
		Name:    "Legacy",
		Prefix:  "legacy",
		BaseURL: "https://legacy.example/v1",
		APIKeys: []string{"provider-secret"},
		Models:  []modelSpec{{Name: "legacy-model"}},
		Enabled: false,
	}, true)
	t.Cleanup(func() { storeProviderSpec(providerSpec{}, false) })

	caps := registrationCapabilities()
	if caps.Executor || caps.ModelRegistrar || caps.ModelProvider || caps.ExecutorModelScope != "" {
		t.Fatalf("direct mode advertised legacy executor/model capabilities: %+v", caps)
	}
	if !caps.RequestInterceptor || !caps.ManagementAPI {
		t.Fatalf("direct mode dropped base capabilities: %+v", caps)
	}
	if response := currentModelResponse(); len(response.Models) != 0 {
		t.Fatalf("direct mode published legacy models: %+v", response.Models)
	}
}

func TestDirectModelCatalogParserBoundsDeduplicatesAndPreservesMetadata(t *testing.T) {
	existing := []directAccountModel{
		{UpstreamID: "a", Alias: "alias-a", Enabled: false, Image: true},
		{UpstreamID: "old", Alias: "alias-old", Enabled: true},
	}
	models := parseDirectModelCatalog([]byte(`{"data":[{"id":"b"},{"id":"a"},{"id":"a"},{"id":"c"}]}`), existing, 3)
	if len(models) != 3 {
		t.Fatalf("model count = %d, want 3: %+v", len(models), models)
	}
	byID := map[string]directAccountModel{}
	for _, model := range models {
		byID[model.UpstreamID] = model
	}
	if byID["a"].Alias != "alias-a" || byID["a"].Enabled || !byID["a"].Image || byID["a"].Unavailable {
		t.Fatalf("existing metadata for a was not preserved: %+v", byID["a"])
	}
	if !byID["b"].Enabled {
		t.Fatalf("new model should default enabled: %+v", byID["b"])
	}
	if _, exists := byID["old"]; exists {
		t.Fatalf("bounded output should not exceed limit with old missing rows: %+v", models)
	}

	fallback := parseDirectModelCatalog([]byte(`["z", {"id":"y"}, "z"]`), nil, 10)
	if len(fallback) != 2 || fallback[0].UpstreamID != "y" || fallback[1].UpstreamID != "z" {
		t.Fatalf("plain-list fallback = %+v", fallback)
	}
}
