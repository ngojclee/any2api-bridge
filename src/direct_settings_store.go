package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

// The direct console used to be read-only plus provider-row upsert, so an
// account list built in the browser disappeared on reload and on every CPA
// restart. These routes give it somewhere to persist:
//
//	POST /direct/accounts/add     create or replace one account by account_id
//	POST /direct/accounts/remove  delete one account by account_id
//	POST /direct/mode             flip direct_mode_enabled
//
// Each call changes one account or one boolean and leaves the rest of the
// stored list alone, so no call here can round-trip away a field an operator
// wrote by hand.

var errInvalidDirectAccountPayload = errors.New("invalid direct account payload")

func handleDirectAccountAdd(request pluginapi.ManagementRequest) ([]byte, error) {
	incoming, errDecode := directAccountFromRequestBody(request.Body)
	if errDecode != nil {
		return managementJSONResponse(http.StatusBadRequest, map[string]string{"error": errDecode.Error()}), nil
	}
	incoming = normalizeDirectAccount(incoming)
	if incoming.AccountID == "" {
		return managementJSONResponse(http.StatusBadRequest, map[string]string{"error": "account_id is required"}), nil
	}
	// validateDirectAccounts reports an absent kind as "unsupported", which is
	// the first thing an operator hits, so name the real problem here.
	if incoming.ProviderKind == "" {
		return managementJSONResponse(http.StatusBadRequest, map[string]string{
			"error": "provider_kind is required: agy2api or gpt2api",
		}), nil
	}
	merged, replaced := mergeDirectAccountByID(currentPluginSettings().DirectAccounts, incoming)
	if errValidate := validateDirectAccounts(merged); errValidate != nil {
		return managementJSONResponse(http.StatusConflict, map[string]string{"error": errValidate.Error()}), nil
	}
	changed, errStore := storeDirectSettings(&merged, nil)
	if errStore != nil {
		return managementJSONResponse(http.StatusInternalServerError, map[string]string{"error": errStore.Error()}), nil
	}
	recordDashboardEvent("success", "Direct account saved to CPA plugin config")
	return managementJSONResponse(http.StatusOK, map[string]any{
		"ok":            true,
		"account_id":    incoming.AccountID,
		"replaced":      replaced,
		"account_count": len(normalizeDirectAccounts(merged)),
		"changed":       changed,
	}), nil
}

func handleDirectAccountRemove(request pluginapi.ManagementRequest) ([]byte, error) {
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
	if accountID == "" {
		return managementJSONResponse(http.StatusBadRequest, map[string]string{"error": "account_id is required"}), nil
	}
	stored := normalizeDirectAccounts(currentPluginSettings().DirectAccounts)
	kept := make([]directAccount, 0, len(stored))
	removed := ""
	for _, account := range stored {
		if strings.EqualFold(account.AccountID, accountID) {
			removed = account.AccountID
			continue
		}
		kept = append(kept, account)
	}
	if removed == "" {
		return managementJSONResponse(http.StatusNotFound, map[string]string{"error": "direct account not found"}), nil
	}
	changed, errStore := storeDirectSettings(&kept, nil)
	if errStore != nil {
		return managementJSONResponse(http.StatusInternalServerError, map[string]string{"error": errStore.Error()}), nil
	}
	recordDashboardEvent("info", "Direct account removed from CPA plugin config")
	return managementJSONResponse(http.StatusOK, map[string]any{
		"ok":            true,
		"removed":       removed,
		"account_count": len(kept),
		"changed":       changed,
	}), nil
}

func handleDirectModeSet(request pluginapi.ManagementRequest) ([]byte, error) {
	var payload struct {
		Enabled *bool `json:"enabled"`
	}
	if errUnmarshal := json.Unmarshal(request.Body, &payload); errUnmarshal != nil {
		return managementJSONResponse(http.StatusBadRequest, map[string]string{"error": "invalid direct mode payload"}), nil
	}
	if payload.Enabled == nil {
		return managementJSONResponse(http.StatusBadRequest, map[string]string{"error": "enabled must be true or false"}), nil
	}
	stored := normalizeDirectAccounts(currentPluginSettings().DirectAccounts)
	if errValidate := validateDirectAccounts(stored); errValidate != nil {
		return managementJSONResponse(http.StatusConflict, map[string]string{
			"error": "stored direct accounts are invalid, fix them before changing direct mode: " + errValidate.Error(),
		}), nil
	}
	if *payload.Enabled && len(stored) == 0 {
		return managementJSONResponse(http.StatusConflict, map[string]string{
			"error": "add at least one direct account before enabling direct mode",
		}), nil
	}
	mode := *payload.Enabled
	changed, errStore := storeDirectSettings(nil, &mode)
	if errStore != nil {
		return managementJSONResponse(http.StatusInternalServerError, map[string]string{"error": errStore.Error()}), nil
	}
	recordDashboardEvent("success", "Direct mode changed in CPA plugin config")
	return managementJSONResponse(http.StatusOK, map[string]any{
		"ok":                  true,
		"direct_mode_enabled": mode,
		"changed":             changed,
	}), nil
}

// directAccountFromRequestBody reads the console draft. A save call can never
// carry a credential: any api_key in the body is dropped, and the key still
// reaches CPA through /direct/accounts/upsert, which owns the provider row.
func directAccountFromRequestBody(raw []byte) (directAccount, error) {
	if len(strings.TrimSpace(string(raw))) == 0 {
		return directAccount{}, errInvalidDirectAccountPayload
	}
	var body map[string]any
	if errUnmarshal := json.Unmarshal(raw, &body); errUnmarshal != nil {
		return directAccount{}, errInvalidDirectAccountPayload
	}
	account := directAccountFromMap(asMap(body))
	account.APIKey = ""
	return account, nil
}

// mergeDirectAccountByID replaces the account with the same account_id, or
// appends a new one. On replace it keeps the stored credential unless the
// caller supplied a new one, and keeps stored static headers when the draft
// carried none. Console drafts never carry credentials; provider imports do.
func mergeDirectAccountByID(stored []directAccount, incoming directAccount) ([]directAccount, bool) {
	out := make([]directAccount, 0, len(stored)+1)
	replaced := false
	for _, account := range stored {
		if !strings.EqualFold(strings.TrimSpace(account.AccountID), incoming.AccountID) {
			out = append(out, account)
			continue
		}
		merged := incoming
		if strings.TrimSpace(merged.APIKey) == "" {
			merged.APIKey = account.APIKey
		}
		// Keep the stored id spelling so a case-insensitive edit cannot quietly
		// rename the account in config and orphan anything referencing it.
		merged.AccountID = account.AccountID
		if len(merged.StaticHeaders) == 0 {
			merged.StaticHeaders = account.StaticHeaders
		}
		if merged.AuthID == "" {
			merged.AuthID = account.AuthID
		}
		out = append(out, normalizeDirectAccount(merged))
		replaced = true
	}
	if !replaced {
		out = append(out, incoming)
	}
	return out, replaced
}
