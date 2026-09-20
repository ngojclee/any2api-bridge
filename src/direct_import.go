package main

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

type directProviderImportView struct {
	Index         int    `json:"index"`
	Name          string `json:"name"`
	Kind          string `json:"kind"`
	BaseURL       string `json:"base_url"`
	Prefix        string `json:"prefix"`
	ModelCount    int    `json:"model_count"`
	KeyConfigured bool   `json:"key_configured"`
	Imported      bool   `json:"imported"`
	Disabled      bool   `json:"disabled"`
}

func handleDirectProviderImportList(request pluginapi.ManagementRequest) ([]byte, error) {
	root, errParse := parseYAMLMap(currentConfigSnapshot().ConfigYAML)
	if errParse != nil {
		return managementJSONResponse(http.StatusBadGateway, map[string]string{
			"error": "CPA config could not be parsed",
		}), nil
	}
	settings := currentPluginSettings()
	imported := make(map[string]struct{}, len(settings.DirectAccounts))
	for _, account := range normalizeDirectAccounts(settings.DirectAccounts) {
		imported[strings.ToLower(strings.TrimSpace(account.ChannelName))] = struct{}{}
	}
	items := make([]directProviderImportView, 0)
	requestedKind := normalizeDirectProviderKind(firstRequestQueryValue(request, "kind"))
	for index, providerMap := range openAICompatEntries(root) {
		name, _ := stringValue(providerMap, "name")
		baseURL, _ := stringValue(providerMap, "base-url", "base_url", "url")
		prefix, _ := stringValue(providerMap, "prefix")
		kind := classifyDirectProviderKind(name, baseURL, prefix)
		if kind == "" {
			continue
		}
		if requestedKind != "" && kind != requestedKind {
			continue
		}
		disabled, _ := boolValue(providerMap, "disabled")
		_, isImported := imported[strings.ToLower(strings.TrimSpace(name))]
		items = append(items, directProviderImportView{
			Index:         index,
			Name:          name,
			Kind:          kind,
			BaseURL:       baseURL,
			Prefix:        prefix,
			ModelCount:    len(compatModels(providerMap)),
			KeyConfigured: len(compatAPIKeys(providerMap)) > 0,
			Imported:      isImported,
			Disabled:      disabled,
		})
	}
	return managementJSONResponse(http.StatusOK, map[string]any{
		"providers": items,
		"count":     len(items),
	}), nil
}

func handleDirectProviderImport(request pluginapi.ManagementRequest) ([]byte, error) {
	var payload struct {
		Index int    `json:"index"`
		Name  string `json:"name"`
	}
	if errUnmarshal := json.Unmarshal(request.Body, &payload); errUnmarshal != nil {
		return managementJSONResponse(http.StatusBadRequest, map[string]string{
			"error": "invalid provider import payload",
		}), nil
	}
	root, errParse := parseYAMLMap(currentConfigSnapshot().ConfigYAML)
	if errParse != nil {
		return managementJSONResponse(http.StatusBadGateway, map[string]string{
			"error": "CPA config could not be parsed",
		}), nil
	}
	entries := openAICompatEntries(root)
	if payload.Index < 0 || payload.Index >= len(entries) {
		return managementJSONResponse(http.StatusNotFound, map[string]string{
			"error": "provider not found",
		}), nil
	}
	providerMap := entries[payload.Index]
	name, _ := stringValue(providerMap, "name")
	baseURL, _ := stringValue(providerMap, "base-url", "base_url", "url")
	prefix, _ := stringValue(providerMap, "prefix")
	kind := classifyDirectProviderKind(name, baseURL, prefix)
	if kind == "" {
		return managementJSONResponse(http.StatusBadRequest, map[string]string{
			"error": "provider is not an Antigravity or ChatGPT provider",
		}), nil
	}
	if strings.TrimSpace(payload.Name) != "" && !strings.EqualFold(strings.TrimSpace(payload.Name), strings.TrimSpace(name)) {
		return managementJSONResponse(http.StatusConflict, map[string]string{
			"error": "provider selection changed; refresh the import list",
		}), nil
	}
	account := directAccountFromProvider(providerMap, kind)
	if strings.TrimSpace(account.AccountID) == "" || strings.TrimSpace(account.ChannelName) == "" {
		return managementJSONResponse(http.StatusBadRequest, map[string]string{
			"error": "provider is missing an importable name",
		}), nil
	}
	merged, replaced := mergeDirectAccountByID(currentPluginSettings().DirectAccounts, account)
	if errValidate := validateDirectAccounts(merged); errValidate != nil {
		return managementJSONResponse(http.StatusConflict, map[string]string{
			"error": errValidate.Error(),
		}), nil
	}
	changed, errStore := storeDirectSettings(&merged, nil)
	if errStore != nil {
		return managementJSONResponse(http.StatusInternalServerError, map[string]string{
			"error": errStore.Error(),
		}), nil
	}
	recordDashboardEvent("success", "Provider imported into direct accounts")
	return managementJSONResponse(http.StatusOK, map[string]any{
		"ok":            true,
		"account_id":    account.AccountID,
		"provider_kind": account.ProviderKind,
		"replaced":      replaced,
		"changed":       changed,
		"account":       directAccountReadView(account),
	}), nil
}

func directAccountFromProvider(providerMap map[string]any, kind string) directAccount {
	name, _ := stringValue(providerMap, "name")
	baseURL, _ := stringValue(providerMap, "base-url", "base_url", "url")
	prefix, _ := stringValue(providerMap, "prefix")
	priority, _ := intValue(providerMap, "priority")
	disabled, _ := boolValue(providerMap, "disabled")
	accountID := slugifyProviderName(name)
	if accountID == "" {
		accountID = kind
	}
	if strings.TrimSpace(prefix) == "" {
		prefix = accountID
	}
	apiKeys := compatAPIKeys(providerMap)
	apiKey := ""
	if len(apiKeys) > 0 {
		apiKey = apiKeys[0]
	}
	models := make([]directAccountModel, 0)
	for _, model := range compatModels(providerMap) {
		models = append(models, directAccountModel{
			UpstreamID:       model.Name,
			Alias:            model.Alias,
			Enabled:          true,
			Image:            model.Image,
			InputModalities:  append([]string(nil), model.InputModalities...),
			OutputModalities: append([]string(nil), model.OutputModalities...),
			Thinking:         model.Thinking,
		})
	}
	return normalizeDirectAccount(directAccount{
		AccountID:              accountID,
		ProviderKind:           kind,
		Label:                  name,
		ChannelName:            name,
		Prefix:                 prefix,
		BaseURL:                baseURL,
		Enabled:                !disabled,
		Priority:               priority,
		IdentitySigningEnabled: true,
		APIKey:                 apiKey,
		Weight:                 1,
		StaticHeaders:          compatHeaders(providerMap),
		Models:                 models,
	})
}

func classifyDirectProviderKind(name, baseURL, prefix string) string {
	haystack := strings.ToLower(strings.Join([]string{name, baseURL, prefix}, " "))
	switch {
	case strings.Contains(haystack, "gpt2api"),
		strings.Contains(haystack, "chatgpt"),
		strings.Contains(haystack, ":8792"),
		strings.Contains(haystack, "chatgpt-web"):
		return directProviderGPT
	case strings.Contains(haystack, "agy2api"),
		strings.Contains(haystack, "antigravity"),
		strings.Contains(haystack, ":8123"):
		return directProviderAGY
	default:
		return ""
	}
}

func slugifyProviderName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	lastDash := false
	for _, char := range value {
		switch {
		case char >= 'a' && char <= 'z', char >= '0' && char <= '9':
			builder.WriteRune(char)
			lastDash = false
		default:
			if !lastDash && builder.Len() > 0 {
				builder.WriteByte('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(builder.String(), "-")
}
