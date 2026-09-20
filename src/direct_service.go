package main

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

type directAccountDetailResponse struct {
	OK      bool              `json:"ok"`
	Account directAccountView `json:"account"`
}

type directScanResult struct {
	OK          bool                      `json:"ok"`
	AccountID   string                    `json:"account_id"`
	HTTPStatus  int                       `json:"http_status"`
	ModelCount  int                       `json:"model_count"`
	Models      []directAccountModelState `json:"models"`
	PreviewOnly bool                      `json:"preview_only"`
	Error       string                    `json:"error,omitempty"`
}

type directPublishResult struct {
	OK                        bool              `json:"ok"`
	AccountID                 string            `json:"account_id"`
	ProviderName              string            `json:"provider_name"`
	ProviderPrefix            string            `json:"provider_prefix"`
	ProviderPayload           map[string]any    `json:"provider_payload"`
	ProviderPayloadConfigured map[string]any    `json:"provider_payload_configured"`
	Changed                   []string          `json:"changed"`
	Validation                map[string]string `json:"validation,omitempty"`
	PreviewOnly               bool              `json:"preview_only"`
}

func handleDirectAccountDetail(request pluginapi.ManagementRequest) ([]byte, error) {
	account, found := directAccountFromRequestID(currentPluginSettings(), request)
	if !found {
		return managementJSONResponse(http.StatusNotFound, map[string]string{"error": "direct account not found"}), nil
	}
	return managementJSONResponse(http.StatusOK, directAccountDetailResponse{
		OK:      true,
		Account: directAccountReadView(account),
	}), nil
}

func handleDirectAccountScan(request pluginapi.ManagementRequest) ([]byte, error) {
	settings := currentPluginSettings()
	account, found := directAccountFromRequestID(settings, request)
	if !found {
		return managementJSONResponse(http.StatusNotFound, map[string]string{"error": "direct account not found"}), nil
	}
	status, models, errProbe := scanDirectAccountModels(settings, account)
	if errProbe != nil {
		return managementJSONResponse(http.StatusBadGateway, directScanResult{
			OK:          false,
			AccountID:   account.AccountID,
			HTTPStatus:  status,
			PreviewOnly: true,
			Error:       errProbe.Error(),
		}), nil
	}
	merged := mergeDirectCatalogSpecs(models, account.Models, maxDirectModels)
	return managementJSONResponse(http.StatusOK, directScanResult{
		OK:          status >= 200 && status < 300,
		AccountID:   account.AccountID,
		HTTPStatus:  status,
		ModelCount:  len(merged),
		Models:      directModelStates(merged),
		PreviewOnly: true,
	}), nil
}

func handleDirectAccountSyncProviderModels(request pluginapi.ManagementRequest) ([]byte, error) {
	settings := currentPluginSettings()
	account, found := directAccountFromRequestID(settings, request)
	if !found {
		return managementJSONResponse(http.StatusNotFound, map[string]string{"error": "direct account not found"}), nil
	}
	root, errParse := parseYAMLMap(currentConfigSnapshot().ConfigYAML)
	if errParse != nil {
		return managementJSONResponse(http.StatusBadGateway, map[string]string{"error": "CPA config could not be parsed"}), nil
	}
	entries := openAICompatEntries(root)
	providerIndex := findDirectProviderIndex(entries, account)
	if providerIndex < 0 {
		return managementJSONResponse(http.StatusConflict, map[string]string{"error": "matching CPA provider was not found"}), nil
	}
	provider := entries[providerIndex]
	providerModels := compatModels(provider)
	if len(providerModels) == 0 {
		return managementJSONResponse(http.StatusConflict, map[string]string{"error": "provider has no configured models"}), nil
	}
	synced := mergeDirectCatalogSpecs(providerModels, account.Models, maxDirectModels)
	active := make([]directAccountModel, 0, len(synced))
	for _, model := range synced {
		if !model.Unavailable {
			active = append(active, model)
		}
	}
	account.Models = normalizeDirectAccountModels(active)
	merged, replaced := mergeDirectAccountByID(settings.DirectAccounts, account)
	if errValidate := validateDirectAccounts(merged); errValidate != nil {
		return managementJSONResponse(http.StatusConflict, map[string]string{"error": errValidate.Error()}), nil
	}
	if _, errStore := storeDirectSettings(&merged, nil); errStore != nil {
		return managementJSONResponse(http.StatusInternalServerError, map[string]string{"error": errStore.Error()}), nil
	}
	providerName, _ := stringValue(provider, "name")
	recordDashboardEvent("success", "Direct account models synced from provider")
	return managementJSONResponse(http.StatusOK, map[string]any{
		"ok":            true,
		"account_id":    account.AccountID,
		"provider_name": providerName,
		"model_count":   len(account.Models),
		"models":        directModelStates(account.Models),
		"replaced":      replaced,
	}), nil
}

func handleDirectAccountPublish(request pluginapi.ManagementRequest) ([]byte, error) {
	account, found := directAccountFromRequestID(currentPluginSettings(), request)
	if !found {
		return managementJSONResponse(http.StatusNotFound, map[string]string{"error": "direct account not found"}), nil
	}
	payload, changed, errPreview := directProviderPreview(currentConfigSnapshot().ConfigYAML, account)
	validation := map[string]string{}
	if account.APIKey == "" {
		validation["api_key"] = "write-only credential is not configured in this draft"
	}
	if errValidate := validateDirectAccounts([]directAccount{account}); errValidate != nil {
		validation["account"] = errValidate.Error()
	}
	if errPreview != nil {
		validation["provider"] = errPreview.Error()
	}
	return managementJSONResponse(http.StatusOK, directPublishResult{
		OK:                        len(validation) == 0,
		AccountID:                 account.AccountID,
		ProviderName:              account.ChannelName,
		ProviderPrefix:            account.Prefix,
		ProviderPayload:           redactDirectProviderPayload(payload),
		ProviderPayloadConfigured: directChannelConfiguredPayload(account),
		Changed:                   changed,
		Validation:                validation,
		PreviewOnly:               true,
	}), nil
}

func directAccountFromRequestID(settings PluginSettings, request pluginapi.ManagementRequest) (directAccount, bool) {
	accountID := strings.TrimSpace(firstRequestQueryValue(request, "account_id", "id"))
	if accountID == "" {
		var payload struct {
			AccountID string `json:"account_id"`
			ID        string `json:"id"`
		}
		if errUnmarshal := json.Unmarshal(request.Body, &payload); errUnmarshal == nil {
			accountID = strings.TrimSpace(firstNonEmpty(payload.AccountID, payload.ID))
		}
	}
	for _, account := range normalizeDirectAccounts(settings.DirectAccounts) {
		if strings.EqualFold(account.AccountID, accountID) ||
			strings.EqualFold(account.ChannelName, accountID) ||
			strings.EqualFold(account.Prefix, accountID) {
			return account, true
		}
	}
	return directAccount{}, false
}

func firstRequestQueryValue(request pluginapi.ManagementRequest, keys ...string) string {
	for _, key := range keys {
		for _, value := range request.Query[key] {
			if strings.TrimSpace(value) != "" {
				return strings.TrimSpace(value)
			}
		}
	}
	if parsed, errParse := url.Parse(request.Path); errParse == nil {
		for _, key := range keys {
			for _, value := range parsed.Query()[key] {
				if strings.TrimSpace(value) != "" {
					return strings.TrimSpace(value)
				}
			}
		}
	}
	return ""
}

func scanDirectAccountModels(settings PluginSettings, account directAccount) (int, []modelSpec, error) {
	spec := providerSpec{
		Name:    account.ChannelName,
		Prefix:  account.Prefix,
		BaseURL: account.BaseURL,
		APIKeys: []string{account.APIKey},
		Headers: account.StaticHeaders,
		Models:  directAccountModelSpecs(account),
	}
	return probeProviderModelSpecs(spec)
}

func mergeDirectCatalogSpecs(models []modelSpec, existing []directAccountModel, limit int) []directAccountModel {
	if limit <= 0 || limit > maxDirectModels {
		limit = maxDirectModels
	}
	existingByID := map[string]directAccountModel{}
	for _, model := range normalizeDirectAccountModels(existing) {
		existingByID[strings.ToLower(model.UpstreamID)] = model
	}
	out := make([]directAccountModel, 0, minInt(limit, len(models)+len(existingByID)))
	seen := map[string]struct{}{}
	for _, spec := range models {
		id := strings.TrimSpace(spec.Name)
		if id == "" {
			continue
		}
		key := strings.ToLower(id)
		model, exists := existingByID[key]
		if !exists {
			model = directAccountModel{UpstreamID: id, Enabled: true}
		}
		model.UpstreamID = id
		model.Unavailable = false
		if spec.Image {
			model.Image = true
		}
		if len(spec.InputModalities) > 0 {
			model.InputModalities = append([]string(nil), spec.InputModalities...)
		}
		if len(spec.OutputModalities) > 0 {
			model.OutputModalities = append([]string(nil), spec.OutputModalities...)
		}
		if spec.Thinking != nil {
			model.Thinking = effectiveThinking(spec.Thinking)
		}
		out = append(out, normalizeDirectAccountModel(model))
		seen[key] = struct{}{}
		if len(out) >= limit {
			break
		}
	}
	if len(out) < limit {
		for _, model := range normalizeDirectAccountModels(existing) {
			key := strings.ToLower(model.UpstreamID)
			if _, exists := seen[key]; exists {
				continue
			}
			model.Unavailable = true
			out = append(out, model)
			if len(out) >= limit {
				break
			}
		}
	}
	return normalizeDirectAccountModels(out)
}

func directModelStates(models []directAccountModel) []directAccountModelState {
	out := make([]directAccountModelState, 0, len(models))
	for _, model := range models {
		out = append(out, directAccountModelState{
			UpstreamID:       model.UpstreamID,
			Alias:            model.Alias,
			Enabled:          model.Enabled,
			Image:            model.Image,
			InputModalities:  append([]string(nil), model.InputModalities...),
			OutputModalities: append([]string(nil), model.OutputModalities...),
			Thinking:         model.Thinking,
			Unavailable:      model.Unavailable,
		})
	}
	return out
}

func redactDirectChannelPayload(payload map[string]any) map[string]any {
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

func directChannelConfiguredPayload(account directAccount) map[string]any {
	headers := map[string]any{}
	for key := range account.StaticHeaders {
		headers[key] = "configured"
	}
	return map[string]any{
		"api_key_configured": account.APIKey != "",
		"headers":            headers,
	}
}

func handleDirectScanBody(request pluginapi.ManagementRequest) ([]byte, error) {
	var payload struct {
		AccountID string               `json:"account_id"`
		Raw       string               `json:"raw"`
		Existing  []directAccountModel `json:"existing"`
		Limit     int                  `json:"limit"`
	}
	if errUnmarshal := json.Unmarshal(request.Body, &payload); errUnmarshal != nil {
		return managementJSONResponse(http.StatusBadRequest, map[string]string{"error": "invalid direct scan payload"}), nil
	}
	models := parseDirectModelCatalog([]byte(payload.Raw), payload.Existing, payload.Limit)
	return managementJSONResponse(http.StatusOK, map[string]any{
		"ok":          true,
		"account_id":  payload.AccountID,
		"model_count": len(models),
		"models":      directModelStates(models),
	}), nil
}
