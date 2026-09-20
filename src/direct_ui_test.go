package main

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

func TestDirectAccountJSONContractRedactsSecrets(t *testing.T) {
	settings := defaultPluginSettings()
	settings.DirectModeEnabled = true
	settings.DirectAccounts = []directAccount{{
		AccountID:              "gpt-prod",
		ProviderKind:           directProviderGPT,
		Label:                  "GPT Production",
		ChannelName:            "GPT Direct",
		Prefix:                 "gpt",
		BaseURL:                "https://gpt2api.example/v1?token=hidden",
		Enabled:                true,
		Priority:               50,
		IdentitySigningEnabled: true,
		APIKey:                 "secret-api-key",
		StaticHeaders:          map[string]string{"X-Static-Secret": "secret-header"},
		Models: []directAccountModel{
			{UpstreamID: "gpt-5.6", Alias: "chatgpt-main", Enabled: true},
		},
	}}
	withSettings(t, settings)

	request := pluginapi.ManagementRequest{
		Path: "/v0/management/plugins/any2api-bridge/direct/accounts/detail",
		Query: url.Values{
			"account_id": {"gpt-prod"},
		},
	}
	raw, errHandle := handleDirectAccountDetail(request)
	if errHandle != nil {
		t.Fatal(errHandle)
	}
	var decoded struct {
		OK     bool `json:"ok"`
		Result struct {
			StatusCode int    `json:"StatusCode"`
			Body       []byte `json:"body"`
		} `json:"result"`
	}
	if errUnmarshal := json.Unmarshal(raw, &decoded); errUnmarshal != nil {
		t.Fatal(errUnmarshal)
	}
	if decoded.Result.StatusCode != 200 {
		t.Fatalf("status = %d body = %s", decoded.Result.StatusCode, decoded.Result.Body)
	}
	for _, secret := range []string{"secret-api-key", "secret-header", "?token=hidden"} {
		if strings.Contains(string(decoded.Result.Body), secret) {
			t.Fatalf("direct account JSON leaked %q: %s", secret, decoded.Result.Body)
		}
	}
	var detail directAccountDetailResponse
	if errUnmarshal := json.Unmarshal(decoded.Result.Body, &detail); errUnmarshal != nil {
		t.Fatal(errUnmarshal)
	}
	if !detail.OK || !detail.Account.APIKeyConfigured || detail.Account.Headers[0].Key != "X-Static-Secret" || !detail.Account.Headers[0].Configured {
		t.Fatalf("redacted account detail missing configured state: %+v", detail)
	}
}

func TestDirectScanBodyParsesAndMergesModelMetadata(t *testing.T) {
	request := pluginapi.ManagementRequest{
		Body: []byte(`{"account_id":"agy-prod","raw":"{\"data\":[{\"id\":\"b\"},{\"id\":\"a\"},{\"id\":\"a\"}]}","existing":[{"upstream_id":"a","alias":"alias-a","enabled":false,"image":true},{"upstream_id":"old","alias":"alias-old","enabled":true}],"limit":10}`),
	}
	raw, errHandle := handleDirectScanBody(request)
	if errHandle != nil {
		t.Fatal(errHandle)
	}
	var decoded struct {
		OK     bool `json:"ok"`
		Result struct {
			StatusCode int    `json:"StatusCode"`
			Body       []byte `json:"body"`
		} `json:"result"`
	}
	if errUnmarshal := json.Unmarshal(raw, &decoded); errUnmarshal != nil {
		t.Fatal(errUnmarshal)
	}
	var body struct {
		OK         bool                      `json:"ok"`
		ModelCount int                       `json:"model_count"`
		Models     []directAccountModelState `json:"models"`
	}
	if errUnmarshal := json.Unmarshal(decoded.Result.Body, &body); errUnmarshal != nil {
		t.Fatal(errUnmarshal)
	}
	if !body.OK || body.ModelCount != 3 {
		t.Fatalf("scan merge = %+v", body)
	}
	byID := map[string]directAccountModelState{}
	for _, model := range body.Models {
		byID[model.UpstreamID] = model
	}
	if byID["a"].Alias != "alias-a" || byID["a"].Enabled || !byID["a"].Image || byID["a"].Unavailable {
		t.Fatalf("existing metadata was not preserved: %+v", byID["a"])
	}
	if !byID["old"].Unavailable {
		t.Fatalf("missing old model should be marked unavailable: %+v", byID["old"])
	}
}

func TestDirectPublishPreviewWritesSelectedModelsToChannelPayload(t *testing.T) {
	previous := currentConfigSnapshot()
	t.Cleanup(func() { applyPluginConfiguration(previous) })
	t.Setenv("CPA_CONFIG_PATH", "Z:\\missing\\cpa-config.yaml")
	applyPluginConfiguration(loadPluginConfiguration([]byte(`
openai-compatibility:
  - name: AGY Direct
    prefix: agy
    base-url: https://agy2api.example/v1
`)))
	settings := defaultPluginSettings()
	settings.DirectModeEnabled = true
	settings.DirectAccounts = []directAccount{{
		AccountID:              "agy-prod",
		ProviderKind:           directProviderAGY,
		ChannelName:            "AGY Direct",
		Prefix:                 "agy",
		BaseURL:                "https://agy2api.example/v1",
		Enabled:                true,
		Priority:               30,
		IdentitySigningEnabled: true,
		APIKey:                 "provider-secret",
		StaticHeaders:          map[string]string{"X-Static": "secret-header"},
		Models: []directAccountModel{
			{UpstreamID: "gemini-3.8-flash", Alias: "agy-fast", Enabled: true},
			{UpstreamID: "hidden", Alias: "off", Enabled: false},
		},
	}}
	withSettings(t, settings)
	raw, errHandle := handleDirectAccountPublish(pluginapi.ManagementRequest{
		Path: "/v0/management/plugins/any2api-bridge/direct/accounts/publish",
		Query: url.Values{
			"account_id": {"agy-prod"},
		},
	})
	if errHandle != nil {
		t.Fatal(errHandle)
	}
	var decoded struct {
		OK     bool `json:"ok"`
		Result struct {
			StatusCode int    `json:"StatusCode"`
			Body       []byte `json:"body"`
		} `json:"result"`
	}
	if errUnmarshal := json.Unmarshal(raw, &decoded); errUnmarshal != nil {
		t.Fatal(errUnmarshal)
	}
	var body directPublishResult
	if errUnmarshal := json.Unmarshal(decoded.Result.Body, &body); errUnmarshal != nil {
		t.Fatal(errUnmarshal)
	}
	if !body.OK || !body.PreviewOnly {
		t.Fatalf("publish preview rejected: %+v", body)
	}
	models, ok := body.ProviderPayload["models"].([]any)
	if !ok || len(models) != 1 {
		t.Fatalf("provider payload models = %#v", body.ProviderPayload["models"])
	}
	first := models[0].(map[string]any)
	if first["name"] != "gemini-3.8-flash" || first["alias"] != "agy-fast" {
		t.Fatalf("selected model/alias not preserved: %#v", first)
	}
	configured, ok := body.ProviderPayloadConfigured["headers"].(map[string]any)
	if !ok || configured["X-Static"] != "configured" || body.ProviderPayloadConfigured["api_key_configured"] != true {
		t.Fatalf("configured secret state missing: %+v", body.ProviderPayloadConfigured)
	}
	for _, secret := range []string{"provider-secret", "secret-header"} {
		if strings.Contains(string(decoded.Result.Body), secret) {
			t.Fatalf("management response leaked %q: %s", secret, decoded.Result.Body)
		}
	}
}

func TestDirectConsolePageHasProviderGroupsViewsAndNoSecrets(t *testing.T) {
	settings := defaultPluginSettings()
	settings.DirectModeEnabled = true
	settings.DirectAccounts = []directAccount{
		{AccountID: "agy-prod", ProviderKind: directProviderAGY, ChannelName: "AGY Direct", Prefix: "agy", Enabled: true, APIKey: "secret-api-key"},
		{AccountID: "gpt-prod", ProviderKind: directProviderGPT, ChannelName: "GPT Direct", Prefix: "gpt", Enabled: true, APIKey: "secret-api-key"},
	}
	withSettings(t, settings)
	page := directAccountConsolePage(scanProviderDiagnostics(), url.Values{})
	for _, expected := range []string{
		"Antigravity / AGY2API",
		"ChatGPT / GPT2API",
		"Accounts",
		"Models",
		"Headers",
		"Routing",
		"Usage",
		"Logs",
		"Settings",
		"/direct/accounts/scan",
		"/direct/accounts/scan/upsert",
		"/direct/accounts/publish",
		"Upsert provider",
		"Direct provider console",
	} {
		if !strings.Contains(page, expected) {
			t.Fatalf("direct console missing %q", expected)
		}
	}
	if strings.Contains(page, "secret-api-key") {
		t.Fatal("direct console leaked API key")
	}
}

func TestDirectModeOffLeavesLegacyMirrorPathUntouched(t *testing.T) {
	loadMirror(t)
	settings := currentPluginSettings()
	settings.DirectModeEnabled = false
	settings.ExecutorEnabled = true
	settings.ModelNamespace = "spike."
	withSettings(t, settings)
	storeProviderSpec(providerSpec{}, false)
	if _, found := resolveProviderSpec(); !found {
		t.Fatal("legacy mirror provider was not resolved")
	}
	caps := registrationCapabilities()
	if !caps.Executor || !caps.ModelRegistrar || !caps.ModelProvider {
		t.Fatalf("legacy mirror capabilities changed while direct mode off: %+v", caps)
	}
	response := currentModelResponse()
	if len(response.Models) != 3 {
		t.Fatalf("legacy model registration changed while direct mode off: %+v", response.Models)
	}
}
