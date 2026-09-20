package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
	"gopkg.in/yaml.v3"
)

func pluginapiRequest(t *testing.T, method, path string, body any) pluginapi.ManagementRequest {
	t.Helper()
	request := pluginapi.ManagementRequest{
		Method: method,
		Path:   managementBasePath + path,
		Query:  url.Values{},
	}
	if body != nil {
		raw, errMarshal := json.Marshal(body)
		if errMarshal != nil {
			t.Fatalf("marshal body: %v", errMarshal)
		}
		request.Body = raw
	}
	return request
}

func TestPatchDirectSettingsConfigWritesAccountsAndPreservesRest(t *testing.T) {
	raw := []byte(`openai-compatibility:
  - name: AGY Direct
    base-url: https://agy2api.example/v1
plugins:
  configs:
    some-other-plugin:
      enabled: true
    ` + pluginID + `:
      enabled: true
      hmac_secret: stored-secret
`)
	accounts := []directAccount{{
		AccountID:              "agy-prod",
		ProviderKind:           directProviderAGY,
		ChannelName:            "AGY Direct",
		Prefix:                 "agy",
		BaseURL:                "https://agy2api.example/v1",
		Enabled:                true,
		Priority:               30,
		IdentitySigningEnabled: true,
		Weight:                 2,
	}}
	out, changed, errPatch := patchDirectSettingsConfig(raw, &accounts, nil)
	if errPatch != nil {
		t.Fatalf("patch: %v", errPatch)
	}
	if !containsString(changed, "direct_accounts") {
		t.Fatalf("expected direct_accounts in changed, got %v", changed)
	}
	root := map[string]any{}
	if errUnmarshal := yaml.Unmarshal(out, &root); errUnmarshal != nil {
		t.Fatalf("unmarshal result: %v", errUnmarshal)
	}
	section := pluginSectionForTest(t, root)
	if _, ok := mapValue(section, "direct_accounts"); !ok {
		t.Fatalf("direct_accounts missing from plugin section: %s", out)
	}
	if text, _ := stringValue(section, "hmac_secret"); text != "stored-secret" {
		t.Fatalf("unrelated plugin key was not preserved: %s", out)
	}
	if _, found := mapValueByNormalizedKey(mustMap(t, root, "plugins", "configs"), "some-other-plugin"); !found {
		t.Fatalf("sibling plugin config was dropped: %s", out)
	}
	if entries := openAICompatEntries(root); len(entries) != 1 {
		t.Fatalf("provider entries changed: %d", len(entries))
	}
}

func TestStoredDirectAccountsRoundTripWithoutLoss(t *testing.T) {
	stored := []directAccount{{
		AccountID:              "agy-prod",
		ProviderKind:           directProviderAGY,
		Label:                  "Antigravity production",
		ChannelName:            "AGY Direct",
		Prefix:                 "agy",
		BaseURL:                "https://agy2api.example/v1",
		Enabled:                true,
		Priority:               30,
		IdentitySigningEnabled: true,
		AuthID:                 "auth-123",
		APIKey:                 "operator-written-key",
		Weight:                 3,
		ProxyURL:               "socks5://proxy.example:1080",
		StaticHeaders:          map[string]string{"X-Tenant": "acme"},
		Models: []directAccountModel{{
			UpstreamID:      "gemini-pro",
			Alias:           "agy-pro",
			Enabled:         true,
			Image:           true,
			InputModalities: []string{"text", "image"},
		}},
	}}
	raw := []byte("plugins:\n  configs:\n    " + pluginID + ":\n      enabled: true\n")
	out, _, errPatch := patchDirectSettingsConfig(raw, &stored, nil)
	if errPatch != nil {
		t.Fatalf("patch: %v", errPatch)
	}
	reloaded := settingsFromMap(defaultPluginSettings(), pluginSectionFromYAML(t, out))
	got := reloaded.DirectAccounts
	if len(got) != 1 {
		t.Fatalf("expected 1 reloaded account, got %d: %s", len(got), out)
	}
	account := got[0]
	if account.APIKey != "operator-written-key" {
		t.Fatalf("api_key was not round-tripped, save would strip an operator secret")
	}
	if account.StaticHeaders["X-Tenant"] != "acme" {
		t.Fatalf("static headers not round-tripped: %+v", account.StaticHeaders)
	}
	if account.AuthID != "auth-123" || account.Weight != 3 || account.Priority != 30 || account.ProxyURL == "" {
		t.Fatalf("scalar fields lost on round trip: %+v", account)
	}
	if len(account.Models) != 1 || account.Models[0].Alias != "agy-pro" || !account.Models[0].Image {
		t.Fatalf("model capabilities lost on round trip: %+v", account.Models)
	}
}

func TestPatchDirectSettingsConfigRejectsInvalidAccounts(t *testing.T) {
	raw := []byte("plugins:\n  configs:\n    " + pluginID + ":\n      enabled: true\n      direct_accounts:\n        - account_id: keep-me\n          provider_kind: agy2api\n          channel_name: Keep\n          prefix: keep\n")
	bad := []directAccount{
		{AccountID: "a", ProviderKind: directProviderAGY, ChannelName: "A", Prefix: "a", Enabled: true},
		{AccountID: "a", ProviderKind: directProviderAGY, ChannelName: "B", Prefix: "b", Enabled: true},
	}
	out, _, errPatch := patchDirectSettingsConfig(raw, &bad, nil)
	if errPatch == nil {
		t.Fatalf("duplicate account_id must be rejected")
	}
	if out != nil {
		t.Fatalf("rejected patch must not return config bytes")
	}
	if !bytes.Contains(raw, []byte("keep-me")) {
		t.Fatalf("test input drifted")
	}
}

func TestPatchDirectSettingsConfigTogglesDirectModeAndClearsAccounts(t *testing.T) {
	raw := []byte("plugins:\n  configs:\n    " + pluginID + ":\n      enabled: true\n      direct_mode_enabled: true\n")
	off := false
	out, changed, errPatch := patchDirectSettingsConfig(raw, nil, &off)
	if errPatch != nil {
		t.Fatalf("toggle mode: %v", errPatch)
	}
	if !containsString(changed, "direct_mode_enabled") {
		t.Fatalf("expected direct_mode_enabled in changed, got %v", changed)
	}
	section := pluginSectionFromYAML(t, out)
	if enabled, _ := boolValue(section, "direct_mode_enabled"); enabled {
		t.Fatalf("direct mode still true after toggle")
	}

	// Seed a real account list, then clear it, so the delete branch is proving
	// that a key an operator can see in the UI actually disappears.
	seed := []directAccount{{
		AccountID: "agy-prod", ProviderKind: directProviderAGY,
		ChannelName: "AGY Direct", Prefix: "agy", Enabled: true,
	}}
	seeding, _, errSeed := patchDirectSettingsConfig(out, &seed, nil)
	if errSeed != nil {
		t.Fatalf("seed accounts: %v", errSeed)
	}
	if _, found := mapValue(pluginSectionFromYAML(t, seeding), "direct_accounts"); !found {
		t.Fatalf("seed did not write direct_accounts: %s", seeding)
	}

	empty := []directAccount{}
	cleared, changedClear, errClear := patchDirectSettingsConfig(seeding, &empty, nil)
	if errClear != nil {
		t.Fatalf("clear accounts: %v", errClear)
	}
	if _, found := mapValue(pluginSectionFromYAML(t, cleared), "direct_accounts"); found {
		t.Fatalf("empty account list should delete the key, not store an empty one")
	}
	if !containsString(changedClear, "direct_accounts") {
		t.Fatalf("expected direct_accounts in changed, got %v", changedClear)
	}
}

func TestMergeDirectAccountByIDPreservesStoredSecrets(t *testing.T) {
	stored := []directAccount{{
		AccountID:     "agy-prod",
		ProviderKind:  directProviderAGY,
		ChannelName:   "AGY Direct",
		Prefix:        "agy",
		Enabled:       true,
		APIKey:        "provider-key",
		AuthID:        "auth-9",
		StaticHeaders: map[string]string{"X-Tenant": "acme"},
	}, {
		AccountID:    "other",
		ProviderKind: directProviderGPT,
		ChannelName:  "GPT Direct",
		Prefix:       "gpt",
		Enabled:      true,
	}}
	edited := directAccount{
		AccountID:   "AGY-PROD",
		Enabled:     false,
		ChannelName: "AGY Direct",
		Prefix:      "agy",
	}
	merged, replaced := mergeDirectAccountByID(stored, normalizeDirectAccount(edited))
	if !replaced {
		t.Fatalf("expected replace")
	}
	if len(merged) != 2 {
		t.Fatalf("expected 2 accounts, got %d", len(merged))
	}
	var target *directAccount
	for index := range merged {
		if merged[index].AccountID == "agy-prod" {
			target = &merged[index]
		}
	}
	if target == nil {
		t.Fatalf("case-insensitive account_id match failed: %+v", merged)
	}
	if target.APIKey != "provider-key" {
		t.Fatalf("saved draft stripped the stored api_key")
	}
	if target.AuthID != "auth-9" {
		t.Fatalf("saved draft cleared auth_id")
	}
	if target.StaticHeaders["X-Tenant"] != "acme" {
		t.Fatalf("saved draft cleared static headers")
	}
	if target.Enabled {
		t.Fatalf("submitted enabled=false should win")
	}
}

func TestMergeDirectAccountByIDRefreshesSuppliedCredential(t *testing.T) {
	stored := []directAccount{{
		AccountID:    "agy-prod",
		ProviderKind: directProviderAGY,
		ChannelName:  "AGY Direct",
		Prefix:       "agy",
		Enabled:      true,
		APIKey:       "old-provider-key",
	}}
	imported := directAccount{
		AccountID:    "agy-prod",
		ProviderKind: directProviderAGY,
		ChannelName:  "AGY Direct",
		Prefix:       "agy",
		Enabled:      true,
		APIKey:       "new-provider-key",
	}
	merged, replaced := mergeDirectAccountByID(stored, normalizeDirectAccount(imported))
	if !replaced {
		t.Fatalf("expected replace")
	}
	if len(merged) != 1 {
		t.Fatalf("expected 1 account, got %d", len(merged))
	}
	if merged[0].APIKey != "new-provider-key" {
		t.Fatalf("server-side import did not refresh the stored api_key")
	}
}

func TestDirectAccountFromRequestBodyDropsAPIKey(t *testing.T) {
	account, errDecode := directAccountFromRequestBody([]byte(`{
		"account_id":"agy-prod","provider_kind":"antigravity","channel_name":"AGY Direct",
		"prefix":"agy","api_key":"smuggled-secret","enabled":true
	}`))
	if errDecode != nil {
		t.Fatalf("decode: %v", errDecode)
	}
	if account.APIKey != "" {
		t.Fatalf("save payload must not carry a credential")
	}
	if account.ProviderKind != directProviderAGY {
		t.Fatalf("provider_kind alias not normalized: %q", account.ProviderKind)
	}
	if _, errEmpty := directAccountFromRequestBody([]byte("   ")); errEmpty == nil {
		t.Fatalf("empty body rejected")
	}
	if _, errBad := directAccountFromRequestBody([]byte("not json")); errBad == nil {
		t.Fatalf("malformed body rejected")
	}
}

func TestHandleDirectAccountAddPersistsAndSurvivesReload(t *testing.T) {
	previous := currentConfigSnapshot()
	t.Cleanup(func() { applyPluginConfiguration(previous) })
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	initial := []byte("openai-compatibility:\n  - name: AGY Direct\n    base-url: https://agy2api.example/v1\nplugins:\n  configs:\n    " + pluginID + ":\n      enabled: true\n      hmac_secret: do-not-echo\n")
	if errWrite := os.WriteFile(path, initial, 0o600); errWrite != nil {
		t.Fatalf("write fixture: %v", errWrite)
	}
	t.Setenv("CPA_CONFIG_PATH", path)
	applyPluginConfiguration(loadPluginConfiguration(initial))

	addRaw, addErr := handleDirectAccountAdd(pluginapiRequest(t, http.MethodPost, "/direct/accounts/add", map[string]any{
		"account_id":    "agy-prod",
		"label":         "Antigravity production",
		"provider_kind": "agy2api",
		"prefix":        "agy",
		"channel_name":  "AGY Direct",
		"api_key":       "browser-must-not-persist",
	}))
	added := decodeManagementReply(t, addRaw, addErr)
	added.statusIs(t, http.StatusOK)
	if added.Body["ok"] != true || added.Body["account_id"] != "agy-prod" {
		t.Fatalf("expected ok add reply, got %s", added.Raw)
	}

	// A draft without provider_kind is the most common operator mistake, and it
	// must say so rather than echoing an empty string back.
	kindlessRaw, kindlessErr := handleDirectAccountAdd(pluginapiRequest(t, http.MethodPost, "/direct/accounts/add", map[string]any{
		"account_id": "agy-no-kind", "prefix": "nok", "channel_name": "AGY Direct",
	}))
	kindless := decodeManagementReply(t, kindlessRaw, kindlessErr)
	kindless.statusIs(t, http.StatusBadRequest)
	if reason, _ := kindless.Body["error"].(string); !strings.Contains(reason, "provider_kind") {
		t.Fatalf("expected provider_kind guidance, got %s", kindless.Raw)
	}

	onDisk, errRead := os.ReadFile(path)
	if errRead != nil {
		t.Fatalf("read config: %v", errRead)
	}
	if strings.Contains(string(onDisk), "browser-must-not-persist") {
		t.Fatalf("api_key from the request body was persisted to config")
	}
	if !strings.Contains(string(onDisk), "agy-prod") {
		t.Fatalf("account was not written to config: %s", onDisk)
	}

	reloaded := loadPluginConfiguration(onDisk)
	if !reloaded.PluginConfigFound {
		t.Fatalf("reloaded config lost the plugin section")
	}
	if text, _ := stringValue(pluginSectionFromYAML(t, onDisk), "hmac_secret"); text != "do-not-echo" {
		t.Fatalf("save clobbered an unrelated plugin key")
	}
	if len(reloaded.Settings.DirectAccounts) != 1 || reloaded.Settings.DirectAccounts[0].AccountID != "agy-prod" {
		t.Fatalf("account did not survive a config reload: %+v", reloaded.Settings.DirectAccounts)
	}
	if got := currentPluginSettings().DirectAccounts; len(got) != 1 {
		t.Fatalf("in-memory settings not refreshed after save")
	}
}

func TestHandleDirectAccountRemoveKeepsOtherAccounts(t *testing.T) {
	previous := currentConfigSnapshot()
	t.Cleanup(func() { applyPluginConfiguration(previous) })
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	initial := []byte("plugins:\n  configs:\n    " + pluginID + ":\n      enabled: true\n      direct_accounts:\n        - account_id: keep-me\n          provider_kind: agy2api\n          channel_name: Keep\n          prefix: keep\n          enabled: true\n          api_key: kept-secret\n        - account_id: drop-me\n          provider_kind: gpt2api\n          channel_name: Drop\n          prefix: drop\n          enabled: true\n")
	if errWrite := os.WriteFile(path, initial, 0o600); errWrite != nil {
		t.Fatalf("write fixture: %v", errWrite)
	}
	t.Setenv("CPA_CONFIG_PATH", path)
	applyPluginConfiguration(loadPluginConfiguration(initial))

	removeRaw, removeErr := handleDirectAccountRemove(pluginapiRequest(t, http.MethodPost, "/direct/accounts/remove", map[string]any{
		"account_id": "drop-me",
	}))
	reply := decodeManagementReply(t, removeRaw, removeErr)
	reply.statusIs(t, http.StatusOK)
	if reply.Body["ok"] != true || reply.Body["removed"] != "drop-me" {
		t.Fatalf("unexpected remove reply: %s", reply.Raw)
	}
	onDisk, errRead := os.ReadFile(path)
	if errRead != nil {
		t.Fatalf("read config: %v", errRead)
	}
	if strings.Contains(string(onDisk), "drop-me") {
		t.Fatalf("account still present after remove: %s", onDisk)
	}
	if !strings.Contains(string(onDisk), "kept-secret") {
		t.Fatalf("removing one account stripped another account's key")
	}
	missingRaw, missingErr := handleDirectAccountRemove(pluginapiRequest(t, http.MethodPost, "/direct/accounts/remove", map[string]any{
		"account_id": "never-existed",
	}))
	missing := decodeManagementReply(t, missingRaw, missingErr)
	missing.statusIs(t, http.StatusNotFound)
	if missing.Body["ok"] == true {
		t.Fatalf("remove of unknown id must not claim success: %s", missing.Raw)
	}
}

func TestHandleDirectModeSetRefusesEmptyAccountList(t *testing.T) {
	previous := currentConfigSnapshot()
	t.Cleanup(func() { applyPluginConfiguration(previous) })
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	initial := []byte("plugins:\n  configs:\n    " + pluginID + ":\n      enabled: true\n      direct_mode_enabled: false\n")
	if errWrite := os.WriteFile(path, initial, 0o600); errWrite != nil {
		t.Fatalf("write fixture: %v", errWrite)
	}
	t.Setenv("CPA_CONFIG_PATH", path)
	applyPluginConfiguration(loadPluginConfiguration(initial))

	refusedRaw, refusedErr := handleDirectModeSet(
		pluginapiRequest(t, http.MethodPost, "/direct/mode", map[string]any{"enabled": true}),
	)
	refused := decodeManagementReply(t, refusedRaw, refusedErr)
	refused.statusIs(t, http.StatusConflict)
	if refused.Body["ok"] == true {
		t.Fatalf("direct mode must not turn on with zero accounts: %s", refused.Raw)
	}
	if reason, _ := refused.Body["error"].(string); !strings.Contains(strings.ToLower(reason), "account") {
		t.Fatalf("expected a clear reason naming accounts, got %s", refused.Raw)
	}
	onDisk, errRead := os.ReadFile(path)
	if errRead != nil {
		t.Fatalf("read config: %v", errRead)
	}
	if strings.Contains(string(onDisk), "direct_mode_enabled: true") {
		t.Fatalf("refused request still wrote config: %s", onDisk)
	}

	allowedRaw, allowedErr := handleDirectModeSet(
		pluginapiRequest(t, http.MethodPost, "/direct/mode", map[string]any{"enabled": false}),
	)
	allowed := decodeManagementReply(t, allowedRaw, allowedErr)
	allowed.statusIs(t, http.StatusOK)
	if allowed.Body["direct_mode_enabled"] != false {
		t.Fatalf("explicit disable should be accepted and echoed: %s", allowed.Raw)
	}
}

func pluginSectionFromYAML(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	root := map[string]any{}
	if errUnmarshal := yaml.Unmarshal(raw, &root); errUnmarshal != nil {
		t.Fatalf("unmarshal: %v (%s)", errUnmarshal, raw)
	}
	return pluginSectionForTest(t, root)
}

func pluginSectionForTest(t *testing.T, root map[string]any) map[string]any {
	t.Helper()
	section, found := findPluginConfig(root)
	if !found {
		t.Fatalf("plugin config section not found")
	}
	return section
}

func mustMap(t *testing.T, root map[string]any, keys ...string) map[string]any {
	t.Helper()
	current := root
	for _, key := range keys {
		next, ok := mapValue(current, key)
		if !ok {
			t.Fatalf("missing key %q", key)
		}
		current = asMap(next)
		if current == nil {
			t.Fatalf("key %q is not a map", key)
		}
	}
	return current
}

// managementReply unwraps the CPA management envelope and reports the inner
// HTTP status plus the decoded JSON body, because the envelope itself always
// says ok:true once the plugin answered.
type managementReply struct {
	Status int
	Body   map[string]any
	Raw    string
}

func decodeManagementReply(t *testing.T, raw []byte, errHandler error) managementReply {
	t.Helper()
	if errHandler != nil {
		t.Fatalf("handler error: %v", errHandler)
	}
	var envelope struct {
		OK     bool `json:"ok"`
		Result struct {
			StatusCode int    `json:"StatusCode"`
			Body       []byte `json:"body"`
		} `json:"result"`
	}
	if errUnmarshal := json.Unmarshal(raw, &envelope); errUnmarshal != nil {
		t.Fatalf("decode envelope: %v (%s)", errUnmarshal, raw)
	}
	reply := managementReply{Status: envelope.Result.StatusCode, Raw: string(envelope.Result.Body)}
	if len(envelope.Result.Body) > 0 {
		if errUnmarshal := json.Unmarshal(envelope.Result.Body, &reply.Body); errUnmarshal != nil {
			t.Fatalf("decode body: %v (%s)", errUnmarshal, reply.Raw)
		}
	}
	return reply
}

func (r managementReply) statusIs(t *testing.T, want int) {
	t.Helper()
	if r.Status != want {
		t.Fatalf("status = %d, want %d (body %s)", r.Status, want, r.Raw)
	}
}

// handleManagementForTest drives the full dispatch table so a route can never
// be advertised to CPA without actually being wired.
func handleManagementForTest(t *testing.T, path, body string) []byte {
	t.Helper()
	raw, errMarshal := json.Marshal(pluginapi.ManagementRequest{
		Method: http.MethodPost,
		Path:   path,
		Body:   []byte(body),
	})
	if errMarshal != nil {
		t.Fatalf("marshal request: %v", errMarshal)
	}
	response, errHandle := handleManagement(raw)
	if errHandle != nil {
		t.Fatalf("dispatch %s: %v", path, errHandle)
	}
	return response
}
func TestManagementRegistrationAdvertisesDirectWriteRoutes(t *testing.T) {
	raw := string(handleManagementRegister())
	for _, want := range []string{
		managementBasePath + "/direct/accounts/add",
		managementBasePath + "/direct/accounts/remove",
		managementBasePath + "/direct/mode",
	} {
		if !strings.Contains(raw, want) {
			t.Fatalf("route %q is not advertised to CPA: %s", want, raw)
		}
	}
}

func TestManagementDispatchReachesEachDirectWriteRoute(t *testing.T) {
	cases := []struct {
		path string
		body string
	}{
		{path: "/direct/accounts/add", body: `{}`},
		{path: "/direct/accounts/remove", body: `{}`},
		{path: "/direct/mode", body: `{}`},
	}
	for _, testCase := range cases {
		response := handleManagementForTest(t, managementBasePath+testCase.path, testCase.body)
		if strings.Contains(string(response), "not found") {
			t.Fatalf("route %s is registered but not dispatched", testCase.path)
		}
	}
}

func TestDirectConsoleExposesPersistenceControls(t *testing.T) {
	settings := defaultPluginSettings()
	settings.DirectAccounts = []directAccount{{
		AccountID: "agy-prod", ProviderKind: directProviderAGY,
		ChannelName: "AGY Direct", Prefix: "agy", Enabled: true,
	}}
	withSettings(t, settings)
	page := directAccountConsolePage(scanProviderDiagnostics(), nil)
	for _, want := range []string{
		`data-action="save-account"`,
		`data-action="remove-account"`,
		`data-action="toggle-mode"`,
		"/direct/accounts/add",
		"/direct/accounts/remove",
		"/direct/mode",
	} {
		if !strings.Contains(page, want) {
			t.Fatalf("direct console is missing %q", want)
		}
	}
	// The copy must not promise that a key is stored, because the handler
	// deliberately drops it.
	if strings.Contains(page, "Draft only. Use Upsert") {
		t.Fatal("console still tells the operator a draft is all the config can hold")
	}
}
