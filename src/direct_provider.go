package main

import (
	"encoding/json"
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
	// Console checkboxes ride along as explicit overrides; they persist onto
	// the stored account through mergeDirectAccountByID + storeDirectSettings
	// below, so the next scan starts from the same choice.
	directFlagOverrides(request, &account)
	status, scanned, errProbe := scanDirectAccountModels(settings, account)
	if errProbe != nil {
		return managementJSONResponse(http.StatusBadGateway, map[string]any{
			"ok":          false,
			"account_id":  account.AccountID,
			"http_status": status,
			"error":       errProbe.Error(),
		}), nil
	}
	if len(scanned) == 0 {
		// An empty 200 would mark every tracked model unavailable and prune
		// the provider's rows; treat it as a scan failure instead.
		return managementJSONResponse(http.StatusBadGateway, map[string]any{
			"ok":          false,
			"account_id":  account.AccountID,
			"http_status": status,
			"error":       "upstream returned an empty model list; provider config left unchanged",
		}), nil
	}
	account.Models = mergeDirectCatalogSpecs(scanned, account.Models, maxDirectModels)
	account.Models = effectiveDirectModelAliases(account, account.Models)
	merged, _ := mergeDirectAccountByID(settings.DirectAccounts, account)
	if errValidate := validateDirectAccounts(merged); errValidate != nil {
		return managementJSONResponse(http.StatusConflict, map[string]string{"error": errValidate.Error()}), nil
	}
	if _, errStore := storeDirectSettings(&merged, nil); errStore != nil {
		return managementJSONResponse(http.StatusInternalServerError, map[string]string{"error": errStore.Error()}), nil
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

// directFlagOverrides applies the model-flag checkboxes. The console posts
// them inside the JSON body while management-API callers may carry them as
// query values, so both channels are checked.
func directFlagOverrides(request pluginapi.ManagementRequest, account *directAccount) {
	value := func(keys ...string) string {
		if query := firstRequestQueryValue(request, keys...); query != "" {
			return query
		}
		var body map[string]any
		if errUnmarshal := json.Unmarshal(request.Body, &body); errUnmarshal != nil || body == nil {
			return ""
		}
		for _, key := range keys {
			switch raw := body[key].(type) {
			case string:
				if trimmed := strings.TrimSpace(raw); trimmed != "" {
					return trimmed
				}
			case bool:
				if raw {
					return "true"
				}
				return "false"
			}
		}
		return ""
	}
	if flag := value("raw_names", "raw-names"); flag != "" {
		account.RawNames = flag == "1" || strings.EqualFold(flag, "true")
	}
	if flag := value("manual_alias", "manual-alias"); flag != "" {
		account.ManualAlias = flag == "1" || strings.EqualFold(flag, "true")
	}
	if flag := value("single_id", "single-id"); flag != "" {
		account.SingleID = flag == "1" || strings.EqualFold(flag, "true")
	}
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

// storeDirectAccounts writes the plugin's account list back into
// plugins.configs.<plugin>.direct_accounts so that accounts survive a CPA
// restart. Before this, the console could only draft accounts in the browser,
// so every reload and every restart emptied the page.
//
// The projection below is intentionally non-lossy: whatever an existing stored
// account already carried (including an operator-written api_key) is written
// back unchanged, so saving one account cannot quietly strip another's
// credentials. The browser console never sends a key, so this path cannot add
// a new secret to the config.
// storeDirectSettings persists the plugin-owned direct knobs back into
// plugins.configs.<plugin>. nil selects "leave this key alone".
func storeDirectSettings(accounts *[]directAccount, directMode *bool) ([]string, error) {
	snapshot := currentConfigSnapshot()
	if !snapshot.ConfigPathFound || strings.TrimSpace(snapshot.ConfigPath) == "" {
		return nil, fmt.Errorf("mounted CPA config path was not found")
	}
	updated, changed, errPatch := patchDirectSettingsConfig(snapshot.ConfigYAML, accounts, directMode)
	if errPatch != nil {
		return nil, errPatch
	}
	if errWrite := writeCPAConfigFile(snapshot.ConfigPath, updated); errWrite != nil {
		return nil, errWrite
	}
	applyPluginConfiguration(loadPluginConfiguration(updated))
	storeProviderSpec(providerSpec{}, false)
	return changed, nil
}

func patchDirectSettingsConfig(raw []byte, accounts *[]directAccount, directMode *bool) ([]byte, []string, error) {
	root, errParse := parseYAMLMap(raw)
	if errParse != nil {
		return nil, nil, fmt.Errorf("parse CPA config: %w", errParse)
	}
	if root == nil {
		return nil, nil, fmt.Errorf("CPA config is empty")
	}
	changed := make([]string, 0, 2)
	pluginConfig := ensurePluginConfig(root)
	if accounts != nil {
		normalized := normalizeDirectAccounts(*accounts)
		if errValidate := validateDirectAccounts(normalized); errValidate != nil {
			return nil, nil, errValidate
		}
		if len(normalized) == 0 {
			deleteNormalized(pluginConfig, "direct_accounts", &changed)
		} else {
			setAny(pluginConfig, "direct_accounts", directAccountsForConfig(normalized), &changed)
		}
	}
	if directMode != nil {
		setBool(pluginConfig, "direct_mode_enabled", *directMode, &changed)
	}
	out, errMarshal := yaml.Marshal(root)
	if errMarshal != nil {
		return nil, nil, fmt.Errorf("marshal CPA config: %w", errMarshal)
	}
	return out, uniqueStrings(changed), nil
}

// directAccountsForConfig projects accounts onto the config shape. api_key and
// header values are deliberately absent so a stored account never duplicates a
// credential that the provider row already owns.
func directAccountsForConfig(accounts []directAccount) []any {
	out := make([]any, 0, len(accounts))
	for _, account := range accounts {
		entry := map[string]any{
			"account_id":               account.AccountID,
			"provider_kind":            account.ProviderKind,
			"channel_name":             account.ChannelName,
			"prefix":                   account.Prefix,
			"enabled":                  account.Enabled,
			"priority":                 account.Priority,
			"identity_signing_enabled": account.IdentitySigningEnabled,
			"weight":                   account.Weight,
		}
		if account.RawNames {
			entry["raw_names"] = true
		}
		if account.ManualAlias {
			entry["manual_alias"] = true
		}
		if account.SingleID {
			entry["single_id"] = true
		}
		if account.Label != "" {
			entry["label"] = account.Label
		}
		if account.BaseURL != "" {
			entry["base_url"] = account.BaseURL
		}
		if account.AuthID != "" {
			entry["auth_id"] = account.AuthID
		}
		if account.APIKey != "" {
			entry["api_key"] = account.APIKey
		}
		if account.ProxyURL != "" {
			entry["proxy_url"] = account.ProxyURL
		}
		if len(account.StaticHeaders) > 0 {
			// Round-trip the map the account already carried rather than a
			// name-only projection: the parser only understands "headers", so
			// anything else would silently drop the operator's config on save.
			headers := make(map[string]any, len(account.StaticHeaders))
			for key, value := range account.StaticHeaders {
				headers[key] = value
			}
			entry["headers"] = headers
		}
		if len(account.Models) > 0 {
			entry["models"] = directAccountModelsForConfig(account.Models)
		}
		out = append(out, entry)
	}
	return out
}

func directAccountModelsForConfig(models []directAccountModel) []any {
	out := make([]any, 0, len(models))
	for _, model := range models {
		entry := map[string]any{
			"upstream_id": model.UpstreamID,
			"enabled":     model.Enabled,
		}
		if model.Alias != "" {
			entry["alias"] = model.Alias
		}
		if model.Image {
			entry["image"] = true
		}
		if len(model.InputModalities) > 0 {
			entry["input_modalities"] = append([]string(nil), model.InputModalities...)
		}
		if len(model.OutputModalities) > 0 {
			entry["output_modalities"] = append([]string(nil), model.OutputModalities...)
		}
		if model.Thinking != nil {
			entry["thinking"] = model.Thinking
		}
		if model.Unavailable {
			entry["unavailable"] = true
		}
		out = append(out, entry)
	}
	return out
}

func applyDirectAccountToProvider(provider map[string]any, account directAccount, changed *[]string) {
	account = normalizeDirectAccount(account)
	// SingleID must also drop the provider prefix: CPA registers a
	// "prefix/<id>" clone for every model while a prefix is set, so one
	// catalog entry per model is impossible until the prefix is removed.
	if account.Prefix == "" || account.SingleID {
		deleteNormalized(provider, "prefix", changed)
	} else {
		setString(provider, "prefix", account.Prefix, changed)
	}
	if account.prioritySet {
		setInt(provider, "priority", account.Priority, changed)
	}
	mergeDirectProviderAPIKeys(provider, account, changed)
	mergeDirectProviderHeaders(provider, account, changed)
	mergeDirectProviderModels(provider, account, changed)
}

func mergeDirectProviderAPIKeys(provider map[string]any, account directAccount, changed *[]string) {
	entries := make([]any, 0)
	updatedExisting := false
	if raw, ok := mapValue(provider, "api-key-entries", "api_key_entries"); ok {
		for _, item := range asSlice(raw) {
			if itemMap := asMap(item); itemMap != nil {
				clone := cloneAnyMap(itemMap)
				if account.APIKey != "" {
					if existingKey, _ := stringValue(clone, "api-key", "api_key", "key"); constantTimeEqual(existingKey, account.APIKey) {
						if account.weightSet {
							setInt(clone, "weight", account.Weight, nil)
						}
						if account.ProxyURL != "" {
							setString(clone, "proxy-url", account.ProxyURL, nil)
						}
						updatedExisting = true
					}
				}
				entries = append(entries, clone)
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
			entry := map[string]any{"api-key": account.APIKey}
			if account.weightSet {
				entry["weight"] = account.Weight
			}
			if account.ProxyURL != "" {
				entry["proxy-url"] = account.ProxyURL
			}
			entries = append(entries, entry)
		}
	}
	if len(entries) > 0 && (!updatedExisting || account.weightSet || account.ProxyURL != "") {
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
	// Upstream ids the latest scan confirmed gone are pruned from the provider
	// row (e.g. a namespace rename leaves dead "chatgpt-web/x" names behind).
	// Only rows this account previously tracked are removed; rows the operator
	// added by hand are left alone.
	unavailable := map[string]bool{}
	for _, model := range account.Models {
		if model.Unavailable && strings.TrimSpace(model.UpstreamID) != "" {
			unavailable[strings.ToLower(strings.TrimSpace(model.UpstreamID))] = true
		}
	}
	rows := make([]any, 0, len(rawModels)+len(account.Models))
	indexByName := map[string]int{}
	for _, item := range rawModels {
		row := asMap(item)
		if row == nil {
			continue
		}
		name, _ := stringValue(row, "name", "id")
		if unavailable[strings.ToLower(strings.TrimSpace(name))] ||
			unavailable[strings.ToLower(trimDirectAliasPrefix(account.Prefix, name))] {
			continue
		}
		clone := cloneAnyMap(row)
		rows = append(rows, clone)
		for _, key := range []string{"name", "id", "alias"} {
			if value, ok := stringValue(clone, key); ok && value != "" {
				indexByName[strings.ToLower(value)] = len(rows) - 1
				if key != "alias" {
					// A row written with a namespaced wire name
					// ("antigravity/x") must still be found by the bare
					// upstream id "x" on the next scan. Exact keys win:
					// the stripped form only fills gaps.
					bare := strings.ToLower(trimDirectAliasPrefix(account.Prefix, value))
					if _, taken := indexByName[bare]; !taken {
						indexByName[bare] = len(rows) - 1
					}
				}
			}
		}
	}
	for _, model := range normalizeDirectAccountModels(account.Models) {
		if !model.Enabled || model.Unavailable || model.UpstreamID == "" {
			continue
		}
		index, exists := indexByName[strings.ToLower(model.UpstreamID)]
		if !exists {
			// RawNames strips the account prefix, so a row written as the
			// bare slug "x" must still match a namespaced upstream id
			// "chatgpt/x" on the next scan.
			index, exists = indexByName[strings.ToLower(trimDirectAliasPrefix(account.Prefix, model.UpstreamID))]
		}
		if !exists || index < 0 || index >= len(rows) {
			rows = append(rows, directAccountModelRow(model, account))
			indexByName[strings.ToLower(model.UpstreamID)] = len(rows) - 1
			continue
		}
		existing := asMap(rows[index])
		if existing == nil {
			existing = map[string]any{}
			rows[index] = existing
		}
		mergeDirectAccountModelRow(existing, model, account)
	}
	if len(rows) > 0 {
		setAny(provider, "models", rows, changed)
	}
}

func directAccountModelRow(model directAccountModel, account directAccount) map[string]any {
	row := map[string]any{"name": model.UpstreamID}
	mergeDirectAccountModelRow(row, model, account)
	return row
}

// CPA re-prefixes a provider row's alias at model registration
// ("prefix/alias"), so the row stores the bare form: an account alias of
// "chatgpt/x" lands as "x", which registers both "x" and "chatgpt/x".
// An existing prefixed alias (written before this rule) is healed back to
// bare; a custom bare alias the operator set by hand is left alone.
func mergeDirectAccountModelRow(row map[string]any, model directAccountModel, account directAccount) {
	if account.RawNames {
		// Bare upstream slug: strip the account namespace so a namespaced
		// upstream id like "chatgpt/x" lands as "x" in the name column.
		row["name"] = trimDirectAliasPrefix(account.Prefix, model.UpstreamID)
	} else {
		row["name"] = directProviderWireName(model.UpstreamID, account.Prefix)
	}
	if account.SingleID {
		// alias=name would leave a "prefix/name" clone behind on providers
		// that set the prefix field (chatgpt/chatgpt/x). Deleting the alias
		// registers exactly one catalog id: the wire name.
		delete(row, "alias")
	} else if !account.ManualAlias {
		alias := strings.TrimSpace(model.Alias)
		if alias == "" {
			alias = strings.TrimSpace(model.UpstreamID)
		}
		alias = trimDirectAliasPrefix(account.Prefix, alias)
		if alias != "" {
			existing, _ := stringValue(row, "alias")
			prefixed := strings.ToLower(strings.Trim(strings.TrimSpace(account.Prefix), "/")) + "/"
			if existing == "" || strings.HasPrefix(strings.ToLower(existing), prefixed) {
				row["alias"] = alias
			}
		}
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

// The provider row's "name" is the wire name CPA sends upstream. It always
// carries the account namespace — "antigravity/x", "chatgpt/x" — so every
// provider advertises a consistent "<backend>/<model>" left column regardless
// of what the upstream /v1/models advertises (agy2api stays bare; gpt2api
// already serves chatgpt/* natively). Upstreams accept the prefixed form.
func directProviderWireName(upstreamID, prefix string) string {
	id := strings.TrimSpace(upstreamID)
	p := strings.Trim(strings.TrimSpace(prefix), "/")
	if p == "" || id == "" || strings.HasPrefix(strings.ToLower(id), strings.ToLower(p)+"/") {
		return id
	}
	return p + "/" + id
}

func trimDirectAliasPrefix(prefix, alias string) string {
	p := strings.ToLower(strings.Trim(strings.TrimSpace(prefix), "/"))
	if p != "" && strings.HasPrefix(strings.ToLower(alias), p+"/") {
		return alias[len(p)+1:]
	}
	return alias
}

func findDirectProviderIndex(entries []map[string]any, account directAccount) int {
	for index, provider := range entries {
		name, _ := stringValue(provider, "name")
		prefix, _ := stringValue(provider, "prefix")
		baseURL, _ := stringValue(provider, "base-url", "base_url", "url")
		if account.ChannelName != "" && !strings.EqualFold(strings.TrimSpace(name), account.ChannelName) {
			continue
		}
		if account.BaseURL != "" && strings.TrimRight(strings.TrimSpace(baseURL), "/") != account.BaseURL {
			continue
		}
		// Prefix is mutable account metadata, not an identity key. Only use it
		// as a fallback when the account did not identify the provider by name
		// or base URL. This lets a prefix-less provider be upgraded in place.
		if account.ChannelName == "" && account.BaseURL == "" && account.Prefix != "" {
			if !strings.EqualFold(strings.Trim(strings.TrimSpace(prefix), "/"), account.Prefix) {
				continue
			}
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
