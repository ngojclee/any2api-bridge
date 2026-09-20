package main

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

func TestDirectProviderImportListsAndImportsConfiguredProviders(t *testing.T) {
	initial := []byte(`
openai-compatibility:
  - name: Antigravity
    base-url: http://10.21.4.101:8123/v1
    api-key-entries:
      - api-key: secret-agy-key
    models:
      - name: gemini-3.8-flash
  - name: ChatGPT
    base-url: http://10.11.1.3:8792/v1
    api-key-entries:
      - api-key: secret-gpt-key
    models:
      - name: gpt-5.6
plugins:
  configs:
    ` + pluginID + `:
      enabled: true
`)
	path := filepath.Join(t.TempDir(), "config.yaml")
	if errWrite := os.WriteFile(path, initial, 0o600); errWrite != nil {
		t.Fatal(errWrite)
	}
	t.Setenv("CPA_CONFIG_PATH", path)
	previous := currentConfigSnapshot()
	t.Cleanup(func() { applyPluginConfiguration(previous) })
	applyPluginConfiguration(loadPluginConfiguration(initial))

	listRaw, errList := handleDirectProviderImportList(pluginapiRequest(t, http.MethodGet, "/direct/providers/import", nil))
	list := decodeManagementReply(t, listRaw, errList)
	list.statusIs(t, http.StatusOK)
	providers, ok := list.Body["providers"].([]any)
	if !ok || len(providers) != 2 {
		t.Fatalf("providers = %#v", list.Body["providers"])
	}
	if strings.Contains(list.Raw, "secret-agy-key") || strings.Contains(list.Raw, "secret-gpt-key") {
		t.Fatal("provider import list leaked an API key")
	}

	filteredRaw, errFiltered := handleDirectProviderImportList(pluginapi.ManagementRequest{
		Path:  "/direct/providers/import",
		Query: url.Values{"kind": {"gpt2api"}},
	})
	filtered := decodeManagementReply(t, filteredRaw, errFiltered)
	filtered.statusIs(t, http.StatusOK)
	filteredProviders, ok := filtered.Body["providers"].([]any)
	if !ok || len(filteredProviders) != 1 {
		t.Fatalf("filtered providers = %#v", filtered.Body["providers"])
	}
	filteredProvider, ok := filteredProviders[0].(map[string]any)
	if !ok || filteredProvider["kind"] != directProviderGPT || filteredProvider["name"] != "ChatGPT" {
		t.Fatalf("filtered provider = %#v", filteredProviders[0])
	}

	importRaw, errImport := handleDirectProviderImport(pluginapiRequest(t, http.MethodPost, "/direct/providers/import", map[string]any{
		"index": 0,
		"name":  "Antigravity",
	}))
	imported := decodeManagementReply(t, importRaw, errImport)
	imported.statusIs(t, http.StatusOK)
	if strings.Contains(imported.Raw, "secret-agy-key") {
		t.Fatal("provider import response leaked an API key")
	}
	accounts := currentPluginSettings().DirectAccounts
	if len(accounts) != 1 {
		t.Fatalf("accounts = %+v", accounts)
	}
	if accounts[0].ProviderKind != directProviderAGY || accounts[0].APIKey != "secret-agy-key" {
		t.Fatalf("imported account = %+v", accounts[0])
	}
	if len(accounts[0].Models) != 1 || accounts[0].Models[0].UpstreamID != "gemini-3.8-flash" {
		t.Fatalf("imported models = %+v", accounts[0].Models)
	}
}
