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
	OK                       bool              `json:"ok"`
	AccountID                string            `json:"account_id"`
	ChannelPayload           map[string]any    `json:"channel_payload"`
	ChannelPayloadConfigured map[string]any    `json:"channel_payload_configured"`
	Validation               map[string]string `json:"validation,omitempty"`
	PreviewOnly              bool              `json:"preview_only"`
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
	merged := parseDirectModelCatalogFromSpecs(models, account.Models, maxDirectModels)
	return managementJSONResponse(http.StatusOK, directScanResult{
		OK:          status >= 200 && status < 300,
		AccountID:   account.AccountID,
		HTTPStatus:  status,
		ModelCount:  len(merged),
		Models:      directModelStates(merged),
		PreviewOnly: true,
	}), nil
}

func handleDirectAccountPublish(request pluginapi.ManagementRequest) ([]byte, error) {
	account, found := directAccountFromRequestID(currentPluginSettings(), request)
	if !found {
		return managementJSONResponse(http.StatusNotFound, map[string]string{"error": "direct account not found"}), nil
	}
	payload := directChannelPayload(account)
	validation := map[string]string{}
	if account.APIKey == "" {
		validation["api_key"] = "write-only credential is not configured in this draft"
	}
	if errValidate := validateDirectAccounts([]directAccount{account}); errValidate != nil {
		validation["account"] = errValidate.Error()
	}
	return managementJSONResponse(http.StatusOK, directPublishResult{
		OK:                       len(validation) == 0,
		AccountID:                account.AccountID,
		ChannelPayload:           redactDirectChannelPayload(payload),
		ChannelPayloadConfigured: directChannelConfiguredPayload(account),
		Validation:               validation,
		PreviewOnly:              true,
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

func parseDirectModelCatalogFromSpecs(models []modelSpec, existing []directAccountModel, limit int) []directAccountModel {
	raw, _ := json.Marshal(map[string]any{
		"data": modelSpecsToCatalogItems(models),
	})
	return parseDirectModelCatalog(raw, existing, limit)
}

func modelSpecsToCatalogItems(models []modelSpec) []map[string]any {
	items := make([]map[string]any, 0, len(models))
	for _, model := range models {
		if strings.TrimSpace(model.Name) == "" {
			continue
		}
		items = append(items, map[string]any{"id": model.Name})
	}
	return items
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
