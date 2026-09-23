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
		Priority:               20,
		IdentitySigningEnabled: true,
		APIKey:                 "new-key",
		Weight:                 5,
		ProxyURL:               "http://proxy.internal:8080",
		StaticHeaders:          map[string]string{"X-New": "new-header"},
		Models: []directAccountModel{
			{UpstreamID: "gemini-3.8-flash", Alias: "gemini-fast", Enabled: true, Image: true},
		},
		weightSet:   true,
		prioritySet: true,
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
	if priority, _ := intValue(provider, "priority"); priority != 20 {
		t.Fatalf("provider priority = %d, want 20", priority)
	}
	rawEntries, _ := mapValue(provider, "api-key-entries")
	newEntry := asMap(asSlice(rawEntries)[1])
	if weight, _ := intValue(newEntry, "weight"); weight != 5 {
		t.Fatalf("new key weight = %d, want 5", weight)
	}
	if proxy, _ := stringValue(newEntry, "proxy-url"); proxy != "http://proxy.internal:8080" {
		t.Fatalf("new key proxy-url = %q", proxy)
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
		Weight:       7,
		ProxyURL:     "http://proxy.example:8080",
		weightSet:    true,
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
	entry := asMap(asSlice(mustMapValue(t, openAICompatEntries(root)[0], "api-key-entries"))[0])
	if weight, _ := intValue(entry, "weight"); weight != 7 {
		t.Fatalf("existing key weight = %d, want 7", weight)
	}
	if proxy, _ := stringValue(entry, "proxy-url"); proxy != "http://proxy.example:8080" {
		t.Fatalf("existing key proxy-url = %q", proxy)
	}
}

func mustMapValue(t *testing.T, raw map[string]any, key string) any {
	t.Helper()
	value, ok := mapValue(raw, key)
	if !ok {
		t.Fatalf("missing %s in %+v", key, raw)
	}
	return value
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

func TestDirectProviderUpsertSetsAndDeletesOriginalPrefix(t *testing.T) {
	raw := []byte(`
openai-compatibility:
  - name: Antigravity
    base-url: https://agy.example/v1
`)
	account := directAccount{
		AccountID:    "agy-prod",
		ProviderKind: directProviderAGY,
		ChannelName:  "Antigravity",
		Prefix:       "agy",
		BaseURL:      "https://agy.example/v1",
		Enabled:      true,
	}
	updated, changed, errPatch := patchDirectProviderConfig(raw, account)
	if errPatch != nil {
		t.Fatal(errPatch)
	}
	root, _ := parseYAMLMap(updated)
	provider := openAICompatEntries(root)[0]
	if prefix, _ := stringValue(provider, "prefix"); prefix != "agy" {
		t.Fatalf("prefix was not set on prefix-less provider: %q", prefix)
	}
	if !containsString(changed, "prefix") {
		t.Fatalf("prefix change was not reported: %#v", changed)
	}

	account.Prefix = ""
	updated, changed, errPatch = patchDirectProviderConfig(updated, account)
	if errPatch != nil {
		t.Fatal(errPatch)
	}
	root, _ = parseYAMLMap(updated)
	provider = openAICompatEntries(root)[0]
	if _, found := mapValue(provider, "prefix"); found {
		t.Fatalf("prefix was not deleted: %+v", provider)
	}
	if !containsString(changed, "prefix") {
		t.Fatalf("prefix deletion was not reported: %#v", changed)
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

func TestDirectProviderAliasStripsAccountPrefix(t *testing.T) {
	raw := []byte(`
openai-compatibility:
  - name: Antigravity
    prefix: antigravity
    base-url: https://agy.example/v1
    api-key-entries:
      - api-key: k
    models:
      - name: gemini-3.8-flash
        alias: antigravity/gemini-3.8-flash
      - name: claude-sonnet-4-6
        alias: sonnet-fast
`)
	account := directAccount{
		AccountID:   "agy-prod",
		ChannelName: "Antigravity",
		Prefix:      "antigravity",
		BaseURL:     "https://agy.example/v1",
		Enabled:     true,
		Models: []directAccountModel{
			{UpstreamID: "gemini-3.8-flash", Alias: "antigravity/gemini-3.8-flash", Enabled: true},
			{UpstreamID: "claude-sonnet-4-6", Alias: "antigravity/claude-sonnet-4-6", Enabled: true},
			{UpstreamID: "gemini-3.7-flash", Alias: "antigravity/gemini-3.7-flash", Enabled: true},
		},
	}
	updated, _, errPatch := patchDirectProviderConfig(raw, account)
	if errPatch != nil {
		t.Fatal(errPatch)
	}
	root, _ := parseYAMLMap(updated)
	models := compatModels(openAICompatEntries(root)[0])
	byName := map[string]modelSpec{}
	for _, m := range models {
		byName[m.Name] = m
	}
	// A previously prefixed alias is healed back to the bare form so CPA's
	// own prefix registration produces both "x" and "antigravity/x".
	if byName["gemini-3.8-flash"].Alias != "gemini-3.8-flash" {
		t.Fatalf("prefixed alias was not healed to bare: %+v", byName["gemini-3.8-flash"])
	}
	// A custom bare alias set by hand is preserved across syncs.
	if byName["claude-sonnet-4-6"].Alias != "sonnet-fast" {
		t.Fatalf("custom bare alias was not preserved: %+v", byName["claude-sonnet-4-6"])
	}
	// New rows get the bare form of the account alias as well.
	if byName["gemini-3.7-flash"].Alias != "gemini-3.7-flash" {
		t.Fatalf("new row alias was not stripped to bare: %+v", byName["gemini-3.7-flash"])
	}
}
