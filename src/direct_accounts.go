package main

import (
	"fmt"
	"sort"
	"strings"
)

const (
	directProviderAGY = "agy2api"
	directProviderGPT = "gpt2api"

	maxDirectAccounts = 64
	maxDirectModels   = 512
	maxDirectWeight   = 1000000
)

type directAccount struct {
	AccountID              string `yaml:"account_id" json:"account_id"`
	ProviderKind           string `yaml:"provider_kind" json:"provider_kind"`
	Label                  string `yaml:"label" json:"label"`
	ChannelName            string `yaml:"channel_name" json:"channel_name"`
	Prefix                 string `yaml:"prefix" json:"prefix"`
	BaseURL                string `yaml:"base_url" json:"base_url"`
	Enabled                bool   `yaml:"enabled" json:"enabled"`
	Priority               int    `yaml:"priority" json:"priority"`
	IdentitySigningEnabled bool   `yaml:"identity_signing_enabled" json:"identity_signing_enabled"`
	// RawNames keeps provider row "name" at the upstream id instead of the
	// namespaced wire name (prefix + "/" + id). ManualAlias leaves the row
	// "alias" to the operator: scan/upsert never writes it. SingleID drops
	// the row alias so CPA registers only the wire name — one catalog id
	// per model, the compat-provider equivalent of "keep original" off.
	RawNames      bool                 `yaml:"raw_names" json:"raw_names"`
	ManualAlias   bool                 `yaml:"manual_alias" json:"manual_alias"`
	SingleID      bool                 `yaml:"single_id" json:"single_id"`
	AuthID        string               `yaml:"auth_id" json:"auth_id"`
	APIKey        string               `yaml:"api_key" json:"api_key"`
	Weight        int                  `yaml:"weight" json:"weight"`
	ProxyURL      string               `yaml:"proxy_url" json:"proxy_url"`
	StaticHeaders map[string]string    `yaml:"headers" json:"headers"`
	Models        []directAccountModel `yaml:"models" json:"models"`
	weightSet     bool
	prioritySet   bool
}

type directAccountModel struct {
	UpstreamID       string        `yaml:"upstream_id" json:"upstream_id"`
	Alias            string        `yaml:"alias" json:"alias"`
	Enabled          bool          `yaml:"enabled" json:"enabled"`
	Image            bool          `yaml:"image" json:"image"`
	InputModalities  []string      `yaml:"input_modalities" json:"input_modalities"`
	OutputModalities []string      `yaml:"output_modalities" json:"output_modalities"`
	Thinking         *thinkingSpec `yaml:"thinking" json:"thinking"`
	Unavailable      bool          `yaml:"unavailable" json:"unavailable"`
}

type directAccountView struct {
	AccountID              string                    `json:"account_id"`
	ProviderKind           string                    `json:"provider_kind"`
	Label                  string                    `json:"label,omitempty"`
	ChannelName            string                    `json:"channel_name"`
	Prefix                 string                    `json:"prefix"`
	BaseURL                string                    `json:"base_url,omitempty"`
	Enabled                bool                      `json:"enabled"`
	Priority               int                       `json:"priority"`
	IdentitySigningEnabled bool                      `json:"identity_signing_enabled"`
	AuthID                 string                    `json:"auth_id,omitempty"`
	APIKeyConfigured       bool                      `json:"api_key_configured"`
	Weight                 int                       `json:"weight"`
	ProxyURLConfigured     bool                      `json:"proxy_url_configured"`
	RawNames               bool                      `json:"raw_names"`
	ManualAlias            bool                      `json:"manual_alias"`
	SingleID               bool                      `json:"single_id"`
	Headers                []directHeaderState       `json:"headers,omitempty"`
	Models                 []directAccountModelState `json:"models,omitempty"`
}

type directHeaderState struct {
	Key        string `json:"key"`
	Configured bool   `json:"configured"`
}

type directAccountModelState struct {
	UpstreamID       string        `json:"upstream_id"`
	Alias            string        `json:"alias,omitempty"`
	Enabled          bool          `json:"enabled"`
	Image            bool          `json:"image,omitempty"`
	InputModalities  []string      `json:"input_modalities,omitempty"`
	OutputModalities []string      `json:"output_modalities,omitempty"`
	Thinking         *thinkingSpec `json:"thinking,omitempty"`
	Unavailable      bool          `json:"unavailable,omitempty"`
}

type directManagementState struct {
	DirectModeEnabled bool                `json:"direct_mode_enabled"`
	AccountCount      int                 `json:"account_count"`
	Accounts          []directAccountView `json:"accounts"`
	Warnings          []string            `json:"warnings,omitempty"`
}

func directAccountsFromAny(raw any) []directAccount {
	items := asSlice(raw)
	if len(items) == 0 {
		return nil
	}
	if len(items) > maxDirectAccounts {
		items = items[:maxDirectAccounts]
	}
	out := make([]directAccount, 0, len(items))
	for _, item := range items {
		if accountMap := asMap(item); accountMap != nil {
			out = append(out, directAccountFromMap(accountMap))
		}
	}
	return normalizeDirectAccounts(out)
}

func directAccountFromMap(raw map[string]any) directAccount {
	account := directAccount{}
	account.AccountID, _ = stringValue(raw, "account_id", "account-id", "id")
	account.ProviderKind, _ = stringValue(raw, "provider_kind", "provider-kind", "kind")
	account.Label, _ = stringValue(raw, "label", "name")
	account.ChannelName, _ = stringValue(raw, "channel_name", "channel-name", "channel", "name")
	account.Prefix, _ = stringValue(raw, "prefix")
	account.BaseURL, _ = stringValue(raw, "base_url", "base-url", "url")
	if priority, ok := intValue(raw, "priority"); ok {
		account.Priority = priority
		account.prioritySet = true
	}
	if enabled, ok := boolValue(raw, "enabled"); ok {
		account.Enabled = enabled
	}
	if signing, ok := boolValue(raw, "identity_signing_enabled", "identity-signing-enabled", "signing_enabled", "signing-enabled"); ok {
		account.IdentitySigningEnabled = signing
	}
	if rawNames, ok := boolValue(raw, "raw_names", "raw-names"); ok {
		account.RawNames = rawNames
	}
	if manualAlias, ok := boolValue(raw, "manual_alias", "manual-alias"); ok {
		account.ManualAlias = manualAlias
	}
	if singleID, ok := boolValue(raw, "single_id", "single-id"); ok {
		account.SingleID = singleID
	}
	account.AuthID, _ = stringValue(raw, "auth_id", "auth-id", "selected_auth_id", "selected-auth-id", "cpa_auth_id", "cpa-auth-id")
	account.APIKey, _ = stringValue(raw, "api_key", "api-key")
	if weight, ok := intValue(raw, "weight"); ok {
		account.Weight = weight
		account.weightSet = true
	}
	account.ProxyURL, _ = stringValue(raw, "proxy_url", "proxy-url")
	account.StaticHeaders = directHeadersFromAny(raw["headers"])
	if value, ok := mapValue(raw, "models"); ok {
		account.Models = directAccountModelsFromAny(value)
	}
	return normalizeDirectAccount(account)
}

func normalizeDirectAccounts(accounts []directAccount) []directAccount {
	if len(accounts) > maxDirectAccounts {
		accounts = accounts[:maxDirectAccounts]
	}
	out := make([]directAccount, 0, len(accounts))
	for _, account := range accounts {
		account = normalizeDirectAccount(account)
		if account.AccountID == "" && account.ChannelName == "" && account.Prefix == "" {
			continue
		}
		out = append(out, account)
	}
	return out
}

func normalizeDirectAccount(account directAccount) directAccount {
	account.AccountID = strings.TrimSpace(account.AccountID)
	account.ProviderKind = normalizeDirectProviderKind(account.ProviderKind)
	account.Label = strings.TrimSpace(account.Label)
	account.ChannelName = strings.TrimSpace(account.ChannelName)
	account.Prefix = strings.Trim(strings.TrimSpace(account.Prefix), "/")
	account.BaseURL = strings.TrimRight(strings.TrimSpace(account.BaseURL), "/")
	account.APIKey = strings.TrimSpace(account.APIKey)
	account.AuthID = strings.TrimSpace(account.AuthID)
	account.ProxyURL = strings.TrimSpace(account.ProxyURL)
	if !account.weightSet && account.Weight == 0 {
		account.Weight = 1
	}
	if !account.IdentitySigningEnabled {
		account.IdentitySigningEnabled = false
	}
	if account.StaticHeaders == nil {
		account.StaticHeaders = map[string]string{}
	}
	cleanHeaders := make(map[string]string, len(account.StaticHeaders))
	for key, value := range account.StaticHeaders {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key != "" && value != "" {
			cleanHeaders[key] = value
		}
	}
	account.StaticHeaders = cleanHeaders
	account.Models = normalizeDirectAccountModels(account.Models)
	return account
}

func normalizeDirectProviderKind(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "agy", "antigravity", "agy2api":
		return directProviderAGY
	case "gpt", "chatgpt", "gpt2api":
		return directProviderGPT
	default:
		return value
	}
}

func validateDirectAccounts(accounts []directAccount) error {
	accounts = normalizeDirectAccounts(accounts)
	accountIDs := map[string]string{}
	activePrefixes := map[string]string{}
	activeChannels := map[string]directAccount{}
	activeAliases := map[string]string{}
	for _, account := range accounts {
		if !isSafeDirectToken(account.AccountID) {
			return fmt.Errorf("direct account %q has an invalid account_id", account.AccountID)
		}
		idKey := strings.ToLower(account.AccountID)
		if prior := accountIDs[idKey]; prior != "" {
			return fmt.Errorf("direct account %q duplicates account_id from %q", account.AccountID, prior)
		}
		accountIDs[idKey] = account.AccountID
		if account.ProviderKind != directProviderAGY && account.ProviderKind != directProviderGPT {
			return fmt.Errorf("direct account %q has unsupported provider_kind %q", account.AccountID, account.ProviderKind)
		}
		if account.ChannelName == "" {
			return fmt.Errorf("direct account %q is missing channel_name", account.AccountID)
		}
		if !isSafeDirectPrefix(account.Prefix) {
			return fmt.Errorf("direct account %q has unsafe prefix %q", account.AccountID, account.Prefix)
		}
		if account.Weight > maxDirectWeight {
			return fmt.Errorf("direct account %q weight %d exceeds maximum %d", account.AccountID, account.Weight, maxDirectWeight)
		}
		if !account.Enabled {
			continue
		}
		prefixKey := strings.ToLower(account.Prefix)
		if priorChannel := activePrefixes[prefixKey]; priorChannel != "" && !strings.EqualFold(priorChannel, account.ChannelName) {
			return fmt.Errorf("direct account %q duplicates active prefix %q from channel %q", account.AccountID, account.Prefix, priorChannel)
		}
		activePrefixes[prefixKey] = account.ChannelName
		channelKey := strings.ToLower(account.ChannelName)
		if prior, exists := activeChannels[channelKey]; exists {
			if prior.ProviderKind != account.ProviderKind {
				return fmt.Errorf("direct account %q collides with %q on channel %q across provider kinds", account.AccountID, prior.AccountID, account.ChannelName)
			}
			// Multiple accounts may share one original provider; each account
			// contributes a separate api-key entry.
		}
		activeChannels[channelKey] = account
		channelModelIDs := map[string]string{}
		for _, model := range account.Models {
			// Unavailable rows are historical records (upstream stopped
			// serving the id); they never reach the provider's model list,
			// so they must not participate in alias uniqueness checks —
			// e.g. after a namespace rename a stale "chatgpt-web/x" keeps
			// the generated alias "chatgpt/x" that now belongs to the live
			// "chatgpt/x" model.
			if !model.Enabled || model.Unavailable || model.UpstreamID == "" {
				continue
			}
			modelID := model.Alias
			if modelID == "" {
				modelID = model.UpstreamID
			}
			modelKey := strings.ToLower(modelID)
			if prior := channelModelIDs[modelKey]; prior != "" {
				return fmt.Errorf("direct account %q has duplicate model id or alias %q from %q", account.AccountID, modelID, prior)
			}
			channelModelIDs[modelKey] = model.UpstreamID
			if priorChannel := activeAliases[modelKey]; priorChannel != "" && !strings.EqualFold(priorChannel, account.ChannelName) {
				return fmt.Errorf("direct account %q has model alias %q already used by channel %q", account.AccountID, modelID, priorChannel)
			}
			activeAliases[modelKey] = account.ChannelName
		}
	}
	return nil
}

func isSafeDirectToken(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 64 {
		return false
	}
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
		default:
			return false
		}
	}
	return true
}

func isSafeDirectPrefix(value string) bool {
	return isSafeDirectToken(value) && !strings.ContainsAny(value, `/\`)
}

func directHeadersFromAny(raw any) map[string]string {
	headerMap := asMap(raw)
	if headerMap == nil {
		return nil
	}
	out := make(map[string]string, len(headerMap))
	for key, value := range headerMap {
		if text, ok := value.(string); ok {
			key = strings.TrimSpace(key)
			text = strings.TrimSpace(text)
			if key != "" && text != "" {
				out[key] = text
			}
		}
	}
	return out
}

func directAccountModelsFromAny(raw any) []directAccountModel {
	items := asSlice(raw)
	if len(items) == 0 {
		return nil
	}
	out := make([]directAccountModel, 0, len(items))
	for _, item := range items {
		if modelMap := asMap(item); modelMap != nil {
			out = append(out, directAccountModelFromMap(modelMap))
		}
	}
	return normalizeDirectAccountModels(out)
}

func directAccountModelFromMap(raw map[string]any) directAccountModel {
	model := directAccountModel{Enabled: true}
	model.UpstreamID, _ = stringValue(raw, "upstream_id", "upstream-id", "name", "id")
	model.Alias, _ = stringValue(raw, "alias")
	if enabled, ok := boolValue(raw, "enabled"); ok {
		model.Enabled = enabled
	}
	model.Image, _ = boolValue(raw, "image")
	model.InputModalities, _ = stringSliceValue(raw, "input_modalities", "input-modalities")
	model.OutputModalities, _ = stringSliceValue(raw, "output_modalities", "output-modalities")
	if rawThinking, exists := mapValue(raw, "thinking"); exists {
		if thinkingMap := asMap(rawThinking); thinkingMap != nil {
			thinking := &thinkingSpec{}
			thinking.Min, _ = intValue(thinkingMap, "min")
			thinking.Max, _ = intValue(thinkingMap, "max")
			thinking.ZeroAllowed, _ = boolValue(thinkingMap, "zero-allowed", "zero_allowed")
			thinking.DynamicAllowed, _ = boolValue(thinkingMap, "dynamic-allowed", "dynamic_allowed")
			thinking.Levels, _ = stringSliceValue(thinkingMap, "levels")
			model.Thinking = effectiveThinking(thinking)
		}
	}
	model.Unavailable, _ = boolValue(raw, "unavailable")
	return normalizeDirectAccountModel(model)
}

func normalizeDirectAccountModels(models []directAccountModel) []directAccountModel {
	if len(models) > maxDirectModels {
		models = models[:maxDirectModels]
	}
	out := make([]directAccountModel, 0, len(models))
	seen := map[string]struct{}{}
	for _, model := range models {
		model = normalizeDirectAccountModel(model)
		if model.UpstreamID == "" {
			continue
		}
		key := strings.ToLower(model.UpstreamID)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, model)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return strings.ToLower(out[i].UpstreamID) < strings.ToLower(out[j].UpstreamID)
	})
	return out
}

func normalizeDirectAccountModel(model directAccountModel) directAccountModel {
	model.UpstreamID = strings.TrimSpace(model.UpstreamID)
	model.Alias = strings.TrimSpace(model.Alias)
	model.InputModalities = normalizeModalities(model.InputModalities)
	model.OutputModalities = normalizeModalities(model.OutputModalities)
	model.Thinking = effectiveThinking(model.Thinking)
	return model
}

func directAccountReadView(account directAccount) directAccountView {
	account = normalizeDirectAccount(account)
	headers := make([]directHeaderState, 0, len(account.StaticHeaders))
	for key := range account.StaticHeaders {
		headers = append(headers, directHeaderState{Key: key, Configured: true})
	}
	sort.SliceStable(headers, func(i, j int) bool {
		return strings.ToLower(headers[i].Key) < strings.ToLower(headers[j].Key)
	})
	models := make([]directAccountModelState, 0, len(account.Models))
	for _, model := range account.Models {
		models = append(models, directAccountModelState{
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
	return directAccountView{
		AccountID:              account.AccountID,
		ProviderKind:           account.ProviderKind,
		Label:                  account.Label,
		ChannelName:            account.ChannelName,
		Prefix:                 account.Prefix,
		BaseURL:                redactURL(account.BaseURL),
		Enabled:                account.Enabled,
		Priority:               account.Priority,
		IdentitySigningEnabled: account.IdentitySigningEnabled,
		AuthID:                 account.AuthID,
		APIKeyConfigured:       account.APIKey != "",
		Weight:                 account.Weight,
		ProxyURLConfigured:     account.ProxyURL != "",
		RawNames:               account.RawNames,
		ManualAlias:            account.ManualAlias,
		SingleID:               account.SingleID,
		Headers:                headers,
		Models:                 models,
	}
}

func directAccountsManagementState(settings PluginSettings) directManagementState {
	settings = normalizeSettings(settings)
	warnings := []string{}
	if errValidate := validateDirectAccounts(settings.DirectAccounts); errValidate != nil {
		warnings = append(warnings, errValidate.Error())
	}
	views := make([]directAccountView, 0, len(settings.DirectAccounts))
	for _, account := range settings.DirectAccounts {
		views = append(views, directAccountReadView(account))
	}
	return directManagementState{
		DirectModeEnabled: settings.DirectModeEnabled,
		AccountCount:      len(views),
		Accounts:          views,
		Warnings:          warnings,
	}
}
