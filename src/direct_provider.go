package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
	"gopkg.in/yaml.v3"
)

type directProviderUpsertResult struct {
	OK              bool              `json:"ok"`
	AccountID       string            `json:"account_id"`
	ProviderName    string            `json:"provider_name"`
	ProviderPrefix  string            `json:"provider_prefix"`
	ProviderPayload map[string]any    `json:"provider_payload"`
	Configured      map[string]any    `json:"configured"`
	Changed         []string          `json:"changed"`
	Validation      map[string]string `json:"validation,omitempty"`
	HTTPStatus      int               `json:"http_status,omitempty"`
	ModelCount      int               `json:"model_count,omitempty"`
	PreviewOnly     bool              `json:"preview_only"`
}

func handleDirectProviderUpsert(request pluginapi.ManagementRequest) ([]byte, error) {
	settings := currentPluginSettings()
	if !settings.DirectModeEnabled {
		return managementJSONResponse(http.StatusConflict, map[string]string{
			"error": "direct_mode_enabled must be true before updating the original provider",
		}), nil
	}
	account, found := directAccountFromRequestID(settings, request)
	if !found {
		return managementJSONResponse(http.StatusNotFound, map[string]string{"error": "direct account not found"}), nil
	}
	snapshot := currentConfigSnapshot()
	if !snapshot.ConfigPathFound || strings.TrimSpace(snapshot.ConfigPath) == "" {
		return managementJSONResponse(http.StatusConflict, map[string]string{"error": "mounted CPA config path was not found"}), nil
	}
	updated, changed, errPatch := patchDirectProviderConfig(snapshot.ConfigYAML, account)
	if errPatch != nil {
		return managementJSONResponse(http.StatusBadRequest, map[string]string{"error": errPatch.Error()}), nil
	}
	if errWrite := writeCPAConfigFile(snapshot.ConfigPath, updated); errWrite != nil {
		return managementJSONResponse(http.StatusInternalServerError, map[string]string{"error": errWrite.Error()}), nil
	}
	applyPluginConfiguration(loadPluginConfiguration(updated))
	storeProviderSpec(providerSpec{}, false)
	payload, _, errPreview := directProviderPreview(updated, account)
	if errPreview != nil {
		return managementJSONResponse(http.StatusInternalServerError, map[string]string{"error": errPreview.Error()}), nil
	}
	recordDashboardEvent("success", "Direct account updated original provider")
	return managementJSONResponse(http.StatusOK, directProviderUpsertResult{
		OK:              true,
		AccountID:       account.AccountID,
		ProviderName:    account.ChannelName,
		ProviderPrefix:  account.Prefix,
		ProviderPayload: redactDirectProviderPayload(payload),
		Configured:      directChannelConfiguredPayload(account),
		Changed:         changed,
		PreviewOnly:     false,
	}), nil
}

func handleDirectProviderScanUpsert(request pluginapi.ManagementRequest) ([]byte, error) {
	settings := currentPluginSettings()
	if !settings.DirectModeEnabled {
		return managementJSONResponse(http.StatusConflict, map[string]string{
			"error": "direct_mode_enabled must be true before updating the original provider",
		}), nil
	}
	account, found := directAccountFromRequestID(settings, request)
	if !found {
		return managementJSONResponse(http.StatusNotFound, map[string]string{"error": "direct account not found"}), nil
	}
	status, scanned, errProbe := scanDirectAccountModels(settings, account)
	if errProbe != nil {
		return managementJSONResponse(http.StatusBadGateway, map[string]any{
			"ok":          false,
			"account_id":  account.AccountID,
			"http_status": status,
			"error":       errProbe.Error(),
		}), nil
	}
	account.Models = mergeDirectCatalogSpecs(scanned, account.Models, maxDirectModels)
	snapshot := currentConfigSnapshot()
	if !snapshot.ConfigPathFound || strings.TrimSpace(snapshot.ConfigPath) == "" {
		return managementJSONResponse(http.StatusConflict, map[string]string{"error": "mounted CPA config path was not found"}), nil
	}
	updated, changed, errPatch := patchDirectProviderConfig(snapshot.ConfigYAML, account)
	if errPatch != nil {
		return managementJSONResponse(http.StatusBadRequest, map[string]string{"error": errPatch.Error()}), nil
	}
	if errWrite := writeCPAConfigFile(snapshot.ConfigPath, updated); errWrite != nil {
		return managementJSONResponse(http.StatusInternalServerError, map[string]string{"error": errWrite.Error()}), nil
	}
	applyPluginConfiguration(loadPluginConfiguration(updated))
	storeProviderSpec(providerSpec{}, false)
	payload, _, errPreview := directProviderPreview(updated, account)
	if errPreview != nil {
		return managementJSONResponse(http.StatusInternalServerError, map[string]string{"error": errPreview.Error()}), nil
	}
	recordDashboardEvent("success", "Direct account scan updated original provider models")
	return managementJSONResponse(http.StatusOK, directProviderUpsertResult{
		OK:              true,
		AccountID:       account.AccountID,
		ProviderName:    account.ChannelName,
		ProviderPrefix:  account.Prefix,
		ProviderPayload: redactDirectProviderPayload(payload),
		Configured:      directChannelConfiguredPayload(account),
		Changed:         changed,
		HTTPStatus:      status,
		ModelCount:      len(account.Models),
		PreviewOnly:     false,
	}), nil
}

func directProviderPreview(raw []byte, account directAccount) (map[string]any, []string, error) {
	root, errParse := parseYAMLMap(raw)
	if errParse != nil {
		return nil, nil, fmt.Errorf("parse CPA config: %w", errParse)
	}
	entries := openAICompatEntries(root)
	if len(entries) == 0 {
		return nil, nil, fmt.Errorf("openai-compatibility providers were not found")
	}
	index := findDirectProviderIndex(entries, account)
	if index < 0 {
		return nil, nil, fmt.Errorf("original provider %q was not found in config", account.ChannelName)
	}
	provider := cloneAnyMap(entries[index])
	changed := make([]string, 0, 4)
	applyDirectAccountToProvider(provider, account, &changed)
	return provider, uniqueStrings(changed), nil
}

func patchDirectProviderConfig(raw []byte, account directAccount) ([]byte, []string, error) {
	root, errParse := parseYAMLMap(raw)
	if errParse != nil {
		return nil, nil, fmt.Errorf("parse CPA config: %w", errParse)
	}
	if root == nil {
		return nil, nil, fmt.Errorf("CPA config is empty")
	}
	entries := openAICompatEntries(root)
	if len(entries) == 0 {
		return nil, nil, fmt.Errorf("openai-compatibility providers were not found")
	}
	index := findDirectProviderIndex(entries, account)
	if index < 0 {
		return nil, nil, fmt.Errorf("original provider %q was not found in config", account.ChannelName)
	}
	provider := entries[index]
	changed := make([]string, 0, 4)
	applyDirectAccountToProvider(provider, account, &changed)
	if errUpdate := replaceOpenAICompatEntry(root, index, provider); errUpdate != nil {
		return nil, nil, errUpdate
	}
	out, errMarshal := yaml.Marshal(root)
	if errMarshal != nil {
		return nil, nil, fmt.Errorf("marshal CPA config: %w", errMarshal)
	}
	return out, uniqueStrings(changed), nil
}

func writeCPAConfigFile(path string, updated []byte) error {
	mode := os.FileMode(0600)
	if info, errStat := os.Stat(path); errStat == nil {
		mode = info.Mode().Perm()
	}
	if errWrite := os.WriteFile(path, updated, mode); errWrite != nil {
		return fmt.Errorf("write CPA config: %w", errWrite)
	}
	return nil
}

func applyDirectAccountToProvider(provider map[string]any, account directAccount, changed *[]string) {
	account = normalizeDirectAccount(account)
	mergeDirectProviderAPIKeys(provider, account, changed)
	mergeDirectProviderHeaders(provider, account, changed)
	mergeDirectProviderModels(provider, account, changed)
}

func mergeDirectProviderAPIKeys(provider map[string]any, account directAccount, changed *[]string) {
	entries := make([]any, 0)
	if raw, ok := mapValue(provider, "api-key-entries", "api_key_entries"); ok {
		for _, item := range asSlice(raw) {
			if itemMap := asMap(item); itemMap != nil {
				entries = append(entries, cloneAnyMap(itemMap))
			}
		}
	}
	if account.APIKey != "" {
		existing := compatAPIKeys(provider)
		duplicate := false
		for _, key := range existing {
			if constantTimeEqual(key, account.APIKey) {
				duplicate = true
				break
			}
		}
		if !duplicate {
			entries = append(entries, map[string]any{"api-key": account.APIKey})
		}
	}
	if len(entries) > 0 {
		setAny(provider, "api-key-entries", entries, changed)
	}
}

func mergeDirectProviderHeaders(provider map[string]any, account directAccount, changed *[]string) {
	headers := compatHeaders(provider)
	if headers == nil {
		headers = map[string]string{}
	}
	for key, value := range account.StaticHeaders {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key != "" && value != "" {
			headers[key] = value
		}
	}
	if len(headers) == 0 {
		return
	}
	payload := make(map[string]any, len(headers))
	for key, value := range headers {
		payload[key] = value
	}
	setAny(provider, "headers", payload, changed)
}

func mergeDirectProviderModels(provider map[string]any, account directAccount, changed *[]string) {
	raw, _ := mapValue(provider, "models")
	rawModels := asSlice(raw)
	rows := make([]any, 0, len(rawModels)+len(account.Models))
	indexByName := map[string]int{}
	for _, item := range rawModels {
		row := asMap(item)
		if row == nil {
			continue
		}
		clone := cloneAnyMap(row)
		rows = append(rows, clone)
		for _, key := range []string{"name", "id", "alias"} {
			if value, ok := stringValue(clone, key); ok && value != "" {
				indexByName[strings.ToLower(value)] = len(rows) - 1
			}
		}
	}
	for _, model := range normalizeDirectAccountModels(account.Models) {
		if !model.Enabled || model.UpstreamID == "" {
			continue
		}
		index, exists := indexByName[strings.ToLower(model.UpstreamID)]
		if !exists || index < 0 || index >= len(rows) {
			rows = append(rows, directAccountModelRow(model))
			indexByName[strings.ToLower(model.UpstreamID)] = len(rows) - 1
			continue
		}
		existing := asMap(rows[index])
		if existing == nil {
			existing = map[string]any{}
			rows[index] = existing
		}
		mergeDirectAccountModelRow(existing, model)
	}
	if len(rows) > 0 {
		setAny(provider, "models", rows, changed)
	}
}

func directAccountModelRow(model directAccountModel) map[string]any {
	row := map[string]any{"name": model.UpstreamID}
	mergeDirectAccountModelRow(row, model)
	return row
}

func mergeDirectAccountModelRow(row map[string]any, model directAccountModel) {
	row["name"] = model.UpstreamID
	if model.Alias != "" {
		row["alias"] = model.Alias
	}
	if model.Image {
		row["image"] = true
	}
	if len(model.InputModalities) > 0 {
		row["input-modalities"] = append([]string(nil), model.InputModalities...)
	}
	if len(model.OutputModalities) > 0 {
		row["output-modalities"] = append([]string(nil), model.OutputModalities...)
	}
	if model.Thinking != nil {
		row["thinking"] = model.Thinking
	}
}

func findDirectProviderIndex(entries []map[string]any, account directAccount) int {
	for index, provider := range entries {
		name, _ := stringValue(provider, "name")
		prefix, _ := stringValue(provider, "prefix")
		baseURL, _ := stringValue(provider, "base-url", "base_url", "url")
		if account.ChannelName != "" && !strings.EqualFold(strings.TrimSpace(name), account.ChannelName) {
			continue
		}
		if account.Prefix != "" && !strings.EqualFold(strings.Trim(strings.TrimSpace(prefix), "/"), account.Prefix) {
			continue
		}
		if account.BaseURL != "" && strings.TrimRight(strings.TrimSpace(baseURL), "/") != account.BaseURL {
			continue
		}
		return index
	}
	return -1
}

func redactDirectProviderPayload(payload map[string]any) map[string]any {
	out := cloneAnyMap(payload)
	delete(out, "api-key-entries")
	if headers, ok := mapValue(out, "headers"); ok {
		headerMap := asMap(headers)
		redacted := make(map[string]any, len(headerMap))
		for key := range headerMap {
			redacted[key] = "configured"
		}
		out["headers"] = redacted
	}
	return out
}
