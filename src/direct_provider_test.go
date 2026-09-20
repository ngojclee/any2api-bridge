package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDirectProviderUpsertUpdatesOriginalProviderOnly(t *testing.T) {
	raw := []byte(`
openai-compatibility:
  - name: Antigravity
    prefix: agy
    base-url: https://agy.example/v1
    api-key-entries:
      - api-key: old-key
    headers:
      X-Existing: keep
    models:
      - name: gemini-3.1-pro
        alias: gemini-pro
`)
	account := directAccount{
		AccountID:              "agy-prod",
		ProviderKind:           directProviderAGY,
		ChannelName:            "Antigravity",
		Prefix:                 "agy",
		BaseURL:                "https://agy.example/v1",
		Enabled:                true,
		IdentitySigningEnabled: true,
		APIKey:                 "new-key",
		StaticHeaders:          map[string]string{"X-New": "new-header"},
		Models: []directAccountModel{
			{UpstreamID: "gemini-3.8-flash", Alias: "gemini-fast", Enabled: true, Image: true},
		},
	}
	updated, changed, errPatch := patchDirectProviderConfig(raw, account)
	if errPatch != nil {
		t.Fatal(errPatch)
	}
	root, errParse := parseYAMLMap(updated)
	if errParse != nil {
		t.Fatal(errParse)
	}
	entries := openAICompatEntries(root)
	if len(entries) != 1 {
		t.Fatalf("provider count = %d, want one original provider", len(entries))
	}
	provider := entries[0]
	if name, _ := stringValue(provider, "name"); name != "Antigravity" {
		t.Fatalf("original provider name changed: %q", name)
	}
	if prefix, _ := stringValue(provider, "prefix"); prefix != "agy" {
		t.Fatalf("original provider prefix changed: %q", prefix)
	}
	keys := compatAPIKeys(provider)
	if len(keys) != 2 || keys[0] != "old-key" || keys[1] != "new-key" {
		t.Fatalf("merged key entries = %#v", keys)
	}
	headers := compatHeaders(provider)
	if headers["X-Existing"] != "keep" || headers["X-New"] != "new-header" {
		t.Fatalf("merged headers = %#v", headers)
	}
	models := compatModels(provider)
	if len(models) != 2 {
		t.Fatalf("merged models = %#v", models)
	}
	byName := map[string]modelSpec{}
	for _, model := range models {
		byName[model.Name] = model
	}
	if byName["gemini-3.1-pro"].Alias != "gemini-pro" {
		t.Fatalf("existing alias was not preserved: %+v", byName["gemini-3.1-pro"])
	}
	if byName["gemini-3.8-flash"].Alias != "gemini-fast" || !byName["gemini-3.8-flash"].Image {
		t.Fatalf("new model row was not merged: %+v", byName["gemini-3.8-flash"])
	}
	if !containsString(changed, "api-key-entries") || !containsString(changed, "models") || !containsString(changed, "headers") {
		t.Fatalf("changed fields = %#v", changed)
	}
	if _, found := findPluginConfig(root); found {
		t.Fatal("direct provider upsert unexpectedly added plugin config")
	}
}

func TestDirectProviderKeyMergeDoesNotDuplicateExistingKey(t *testing.T) {
	raw := []byte(`
openai-compatibility:
  - name: ChatGPT
    prefix: gpt
    base-url: https://gpt.example/v1
    api-key-entries:
      - api-key: existing-key
`)
	account := directAccount{
		AccountID:    "gpt-prod",
		ProviderKind: directProviderGPT,
		ChannelName:  "ChatGPT",
		Prefix:       "gpt",
		BaseURL:      "https://gpt.example/v1",
		Enabled:      true,
		APIKey:       "existing-key",
	}
	updated, _, errPatch := patchDirectProviderConfig(raw, account)
	if errPatch != nil {
		t.Fatal(errPatch)
	}
	root, _ := parseYAMLMap(updated)
	keys := compatAPIKeys(openAICompatEntries(root)[0])
	if len(keys) != 1 || keys[0] != "existing-key" {
		t.Fatalf("duplicate key merge = %#v", keys)
	}
}

func TestDirectProviderPreviewRedactsSecretsAndUsesProviderPayload(t *testing.T) {
	raw := []byte(`
openai-compatibility:
  - name: Antigravity
    prefix: agy
    base-url: https://agy.example/v1
    api-key-entries:
      - api-key: old-key
    headers:
      X-Existing: keep
`)
	account := directAccount{
		AccountID:     "agy-prod",
		ProviderKind:  directProviderAGY,
		ChannelName:   "Antigravity",
		Prefix:        "agy",
		BaseURL:       "https://agy.example/v1",
		Enabled:       true,
		APIKey:        "new-secret-key",
		StaticHeaders: map[string]string{"X-New": "new-secret-header"},
	}
	payload, changed, errPreview := directProviderPreview(raw, account)
	if errPreview != nil {
		t.Fatal(errPreview)
	}
	rawPreview, _ := json.Marshal(redactDirectProviderPayload(payload))
	for _, secret := range []string{"old-key", "new-secret-key", "keep", "new-secret-header"} {
		if strings.Contains(string(rawPreview), secret) {
			t.Fatalf("provider preview leaked %q: %s", secret, rawPreview)
		}
	}
	if len(changed) == 0 {
		t.Fatal("provider preview reported no changes")
	}
}

func TestMergeDirectCatalogSpecsPreservesAliasesAndCapabilities(t *testing.T) {
	existing := []directAccountModel{
		{UpstreamID: "a", Alias: "alias-a", Enabled: false},
		{UpstreamID: "old", Alias: "alias-old", Enabled: true},
	}
	upstream := []modelSpec{
		{
			Name:             "a",
			Image:            true,
			InputModalities:  []string{"text", "image"},
			OutputModalities: []string{"image"},
			Thinking:         &thinkingSpec{Levels: []string{"low", "high"}},
		},
		{Name: "b", Thinking: &thinkingSpec{Levels: []string{"none", "medium"}}},
	}
	models := mergeDirectCatalogSpecs(upstream, existing, 10)
	byID := map[string]directAccountModel{}
	for _, model := range models {
		byID[model.UpstreamID] = model
	}
	a := byID["a"]
	if a.Alias != "alias-a" || a.Enabled || !a.Image || a.Unavailable {
		t.Fatalf("existing alias/enabled was not preserved: %+v", a)
	}
	if strings.Join(a.InputModalities, ",") != "text,image" || strings.Join(a.OutputModalities, ",") != "image" {
		t.Fatalf("upstream modalities were not preserved: %+v", a)
	}
	if a.Thinking == nil || strings.Join(a.Thinking.Levels, ",") != "low,high" {
		t.Fatalf("upstream thinking capability was not preserved: %+v", a.Thinking)
	}
	if b := byID["b"]; !b.Enabled || b.Thinking == nil || strings.Join(b.Thinking.Levels, ",") != "none,medium" {
		t.Fatalf("new upstream model capability was not preserved: %+v", b)
	}
	if old := byID["old"]; !old.Unavailable {
		t.Fatalf("missing model should be marked unavailable: %+v", old)
	}
}

func TestDirectMultiAccountResolutionUsesSelectedAuthID(t *testing.T) {
	settings := defaultPluginSettings()
	settings.DirectModeEnabled = true
	settings.DirectAccounts = []directAccount{
		{AccountID: "agy-a", ProviderKind: directProviderAGY, ChannelName: "Antigravity", Prefix: "agy", AuthID: "auth-a", Enabled: true},
		{AccountID: "agy-b", ProviderKind: directProviderAGY, ChannelName: "Antigravity", Prefix: "agy", AuthID: "auth-b", Enabled: true},
	}
	payload := InterceptRequestPayload{
		Model:          "agy/gemini-3.8-flash",
		RequestedModel: "agy/gemini-3.8-flash",
		Metadata:       map[string]any{"selected_auth_id": "auth-b"},
	}
	account, found := resolveDirectAccountForPayload(payload, settings)
	if !found || account.AccountID != "agy-b" {
		t.Fatalf("selected-auth resolution = %+v found=%v", account, found)
	}
}

func TestDirectProviderUpsertDoesNotApplyModelNamespacePrefixStripping(t *testing.T) {
	raw := []byte(`
openai-compatibility:
  - name: Antigravity
    prefix: agy
    base-url: https://agy.example/v1
    models:
      - name: gemini-3.1-pro
`)
	account := directAccount{
		AccountID:    "agy-prod",
		ProviderKind: directProviderAGY,
		ChannelName:  "Antigravity",
		Prefix:       "agy",
		BaseURL:      "https://agy.example/v1",
		Enabled:      true,
		Models: []directAccountModel{
			{UpstreamID: "gemini-3.8-flash", Enabled: true},
		},
	}
	updated, _, errPatch := patchDirectProviderConfig(raw, account)
	if errPatch != nil {
		t.Fatal(errPatch)
	}
	root, _ := parseYAMLMap(updated)
	provider := openAICompatEntries(root)[0]
	if prefix, _ := stringValue(provider, "prefix"); prefix != "agy" {
		t.Fatalf("prefix changed in direct provider mode: %q", prefix)
	}
	models := compatModels(provider)
	if len(models) != 2 || models[0].Name != "gemini-3.1-pro" || models[1].Name != "gemini-3.8-flash" {
		t.Fatalf("direct provider model IDs were rewritten: %#v", models)
	}
	if _, found := findPluginConfig(root); found {
		t.Fatal("direct provider upsert wrote model_namespace/plugin config")
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
