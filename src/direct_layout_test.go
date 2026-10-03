package main

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// CPA v8 groups credential providers under api-keys (see config.example.yaml
// in the running image): api-keys.openai-compatibility is a list whose rows
// carry "keys" instead of "api-key-entries". The legacy top-level layout must
// keep working too, and writes must land in whichever section the row came
// from — writing into the wrong place or with the wrong field name makes the
// strict decoder reject the whole config.

func nestedCompatFixture() []byte {
	return []byte(`api-keys:
  openai-compatibility:
    - name: Antigravity
      base-url: http://10.0.0.1:8123/v1
      keys:
        - api-key: kept-credential
      models:
        - name: stale-model
          alias: agy/stale-model
        - name: kept-model
          alias: agy/kept-model
    - name: ChatGPT
      base-url: http://10.0.0.1:8792/v1
      keys:
        - api-key: chat-key
      models:
        - name: gpt-model
plugins:
  configs:
    ` + pluginID + `:
      enabled: true
`)
}

func TestOpenAICompatSectionFindsNestedProviders(t *testing.T) {
	root, errParse := parseYAMLMap(nestedCompatFixture())
	if errParse != nil {
		t.Fatalf("parse: %v", errParse)
	}
	entries := openAICompatEntries(root)
	if len(entries) != 2 {
		t.Fatalf("expected 2 nested providers, got %d", len(entries))
	}
	name, _ := stringValue(entries[0], "name")
	if name != "Antigravity" {
		t.Fatalf("first provider name = %q", name)
	}
	keys := compatAPIKeys(entries[0])
	if len(keys) != 1 || keys[0] != "kept-credential" {
		t.Fatalf("compatAPIKeys did not read the v8 keys list: %v", keys)
	}
}

func TestOpenAICompatSectionPrefersNestedOverLegacy(t *testing.T) {
	raw := []byte(`api-keys:
  openai-compatibility:
    - name: Nested
      base-url: http://n.example/v1
openai-compatibility:
  - name: Legacy
    base-url: http://l.example/v1
`)
	root, errParse := parseYAMLMap(raw)
	if errParse != nil {
		t.Fatalf("parse: %v", errParse)
	}
	entries := openAICompatEntries(root)
	if len(entries) != 1 {
		t.Fatalf("expected the nested section to win, got %d entries", len(entries))
	}
	name, _ := stringValue(entries[0], "name")
	if name != "Nested" {
		t.Fatalf("legacy section shadowed the v8 one: %q", name)
	}
}

func TestOpenAICompatSectionFallsBackToLegacyRoot(t *testing.T) {
	raw := []byte(`openai-compatibility:
  - name: Legacy
    base-url: http://l.example/v1
`)
	root, errParse := parseYAMLMap(raw)
	if errParse != nil {
		t.Fatalf("parse: %v", errParse)
	}
	entries := openAICompatEntries(root)
	if len(entries) != 1 {
		t.Fatalf("legacy root list not found: %d", len(entries))
	}
}

func TestPatchWritesBackIntoNestedSectionWithKeysField(t *testing.T) {
	account := directAccount{
		AccountID:   "agy",
		ChannelName: "Antigravity",
		BaseURL:     "http://10.0.0.1:8123/v1",
		APIKey:      "new-credential",
		Models: []directAccountModel{
			{UpstreamID: "kept-model", Enabled: true},
			{UpstreamID: "fresh-model", Enabled: true},
		},
	}
	updated, _, errPatch := patchDirectProviderConfig(nestedCompatFixture(), account, nil)
	if errPatch != nil {
		t.Fatalf("patch: %v", errPatch)
	}
	root, errParse := parseYAMLMap(updated)
	if errParse != nil {
		t.Fatalf("reparse: %v", errParse)
	}
	// The provider list must stay nested; a new top-level section would split
	// the config in two and the strict v8 decoder wins the nested one.
	if raw, exists := mapValue(root, "openai-compatibility", "openai_compatibility"); exists && len(asSlice(raw)) > 0 {
		t.Fatalf("patch created a shadow top-level openai-compatibility list")
	}
	entries := openAICompatEntries(root)
	if len(entries) != 2 {
		t.Fatalf("nested provider count changed: %d", len(entries))
	}
	provider := entries[0]
	if _, legacy := provider["api-key-entries"]; legacy {
		t.Fatalf("v8 row carries the legacy api-key-entries field")
	}
	keys := compatAPIKeys(provider)
	if len(keys) != 2 {
		t.Fatalf("expected kept + new credentials, got %v", keys)
	}
	models := compatModels(provider)
	if len(models) != 3 {
		t.Fatalf("expected kept + stale + fresh rows minus none: %v", models)
	}
	// Sibling provider untouched.
	chatName, _ := stringValue(entries[1], "name")
	if chatName != "ChatGPT" {
		t.Fatalf("sibling provider was reordered or renamed: %q", chatName)
	}
}

func TestPatchLegacyLayoutKeepsAPIKeyEntries(t *testing.T) {
	raw := []byte(`openai-compatibility:
  - name: Antigravity
    base-url: http://10.0.0.1:8123/v1
    api-key-entries:
      - api-key: kept-credential
`)
	account := directAccount{
		AccountID:   "agy",
		ChannelName: "Antigravity",
		Models:      []directAccountModel{{UpstreamID: "m", Enabled: true}},
	}
	updated, _, errPatch := patchDirectProviderConfig(raw, account, nil)
	if errPatch != nil {
		t.Fatalf("patch: %v", errPatch)
	}
	root, errParse := parseYAMLMap(updated)
	if errParse != nil {
		t.Fatalf("reparse: %v", errParse)
	}
	provider := openAICompatEntries(root)[0]
	if _, v8 := provider["keys"]; v8 {
		t.Fatalf("legacy row was rewritten with the v8 keys field")
	}
	if keys := compatAPIKeys(provider); len(keys) != 1 {
		t.Fatalf("credential list lost: %v", keys)
	}
}

func TestFindDirectProviderIndexMatchesNameAndBaseURL(t *testing.T) {
	root, errParse := parseYAMLMap(nestedCompatFixture())
	if errParse != nil {
		t.Fatalf("parse: %v", errParse)
	}
	entries := openAICompatEntries(root)
	if idx := findDirectProviderIndex(entries, directAccount{ChannelName: "antigravity"}); idx != 0 {
		t.Fatalf("case-insensitive name match failed: %d", idx)
	}
	if idx := findDirectProviderIndex(entries, directAccount{ChannelName: "ChatGPT"}); idx != 1 {
		t.Fatalf("second provider not matched by name: %d", idx)
	}
	// Trailing slash on the stored base URL must not break the match.
	if idx := findDirectProviderIndex(entries, directAccount{
		ChannelName: "Antigravity",
		BaseURL:     "http://10.0.0.1:8123/v1/",
	}); idx != 0 {
		t.Fatalf("trailing-slash base URL did not match: %d", idx)
	}
	if idx := findDirectProviderIndex(entries, directAccount{ChannelName: "Missing"}); idx != -1 {
		t.Fatalf("unknown name must not match: %d", idx)
	}
}

func TestPruneDirectProviderModelRows(t *testing.T) {
	root, errParse := parseYAMLMap(nestedCompatFixture())
	if errParse != nil {
		t.Fatalf("parse: %v", errParse)
	}
	provider := openAICompatEntries(root)[0]
	if !pruneDirectProviderModelRows(provider, "agy", []string{"stale-model"}) {
		t.Fatalf("stale row was not pruned")
	}
	models := compatModels(provider)
	if len(models) != 1 || models[0].Name != "kept-model" {
		t.Fatalf("wrong rows survived the prune: %v", models)
	}
	if pruneDirectProviderModelRows(provider, "agy", []string{"not-there"}) {
		t.Fatalf("prune of an unknown id reported a change")
	}
	// A namespaced upstream id matches the provider row by its bare form.
	provider = openAICompatEntries(root)[0]
	if !pruneDirectProviderModelRows(provider, "agy", []string{"agy/kept-model"}) {
		t.Fatalf("prefixed id did not prune the bare row")
	}
	if models := compatModels(provider); len(models) != 0 {
		t.Fatalf("expected an empty models list after pruning both rows: %v", models)
	}
}

func TestHandleDirectAccountModelRemovePrunesStoreAndProvider(t *testing.T) {
	previous := currentConfigSnapshot()
	t.Cleanup(func() { applyPluginConfiguration(previous) })
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	initial := []byte(`api-keys:
  openai-compatibility:
    - name: Antigravity
      base-url: http://10.0.0.1:8123/v1
      keys:
        - api-key: kept-credential
      models:
        - name: stale-model
        - name: kept-model
plugins:
  configs:
    ` + pluginID + `:
      enabled: true
      direct_accounts:
        - account_id: agy
          provider_kind: agy2api
          channel_name: Antigravity
          prefix: agy
          base_url: http://10.0.0.1:8123/v1
          enabled: true
          models:
            - upstream_id: stale-model
              enabled: true
              unavailable: true
            - upstream_id: kept-model
              enabled: true
`)
	if errWrite := os.WriteFile(path, initial, 0o600); errWrite != nil {
		t.Fatalf("write fixture: %v", errWrite)
	}
	t.Setenv("CPA_CONFIG_PATH", path)
	applyPluginConfiguration(loadPluginConfiguration(initial))

	raw, errHandle := handleDirectAccountModelRemove(pluginapiRequest(t, http.MethodPost, "/direct/accounts/models/remove", map[string]any{
		"account_id":  "agy",
		"upstream_id": "stale-model",
	}))
	reply := decodeManagementReply(t, raw, errHandle)
	reply.statusIs(t, http.StatusOK)
	if reply.Body["removed"] != "stale-model" || reply.Body["provider_written"] != true {
		t.Fatalf("unexpected remove reply: %s", reply.Raw)
	}

	onDisk, errRead := os.ReadFile(path)
	if errRead != nil {
		t.Fatalf("read config: %v", errRead)
	}
	if strings.Contains(string(onDisk), "stale-model") {
		t.Fatalf("stale model survived in store or provider: %s", onDisk)
	}
	if !strings.Contains(string(onDisk), "kept-model") || !strings.Contains(string(onDisk), "kept-credential") {
		t.Fatalf("remove stripped unrelated data: %s", onDisk)
	}
	root, errParse := parseYAMLMap(onDisk)
	if errParse != nil {
		t.Fatalf("reparse: %v", errParse)
	}
	if entries := openAICompatEntries(root); len(entries) != 1 {
		t.Fatalf("provider list did not stay nested: %d", len(entries))
	}

	missingRaw, missingErr := handleDirectAccountModelRemove(pluginapiRequest(t, http.MethodPost, "/direct/accounts/models/remove", map[string]any{
		"account_id":  "agy",
		"upstream_id": "never-tracked",
	}))
	missing := decodeManagementReply(t, missingRaw, missingErr)
	missing.statusIs(t, http.StatusNotFound)
}
