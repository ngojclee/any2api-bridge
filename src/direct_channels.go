package main

import (
	"encoding/json"
	"strings"
)

func resolveDirectAccountForPayload(payload InterceptRequestPayload, settings PluginSettings) (directAccount, bool) {
	settings = normalizeSettings(settings)
	if !settings.DirectModeEnabled {
		return directAccount{}, false
	}
	candidate := candidateFromPayload(payload, settings)
	if candidate.AuthID != "" {
		for _, account := range settings.DirectAccounts {
			account = normalizeDirectAccount(account)
			if !account.Enabled || account.AuthID == "" {
				continue
			}
			if strings.EqualFold(account.AuthID, candidate.AuthID) {
				return account, true
			}
		}
	}
	for _, account := range settings.DirectAccounts {
		account = normalizeDirectAccount(account)
		if !account.Enabled {
			continue
		}
		if directAccountMatchesCandidate(account, candidate) ||
			modelHasDirectPrefix(payload.RequestedModel, account.Prefix) ||
			modelHasDirectPrefix(payload.Model, account.Prefix) {
			return account, true
		}
	}
	return directAccount{}, false
}

func directAccountMatchesCandidate(account directAccount, candidate providerCandidate) bool {
	if account.AuthID != "" && candidate.AuthID != "" && strings.EqualFold(account.AuthID, candidate.AuthID) {
		return true
	}
	for _, value := range []string{candidate.Name, candidate.ProviderKey, candidate.ResolvedPrefix} {
		if strings.EqualFold(strings.TrimSpace(value), account.ChannelName) ||
			strings.EqualFold(strings.TrimSpace(value), account.Prefix) ||
			strings.EqualFold(strings.TrimSpace(value), account.AccountID) {
			return true
		}
	}
	return false
}

func modelHasDirectPrefix(model, prefix string) bool {
	model = strings.ToLower(strings.TrimSpace(model))
	prefix = strings.ToLower(strings.Trim(strings.TrimSpace(prefix), "/"))
	return model != "" && prefix != "" && strings.HasPrefix(model, prefix+"/")
}

func directAccountProviderCandidate(account directAccount) providerCandidate {
	return providerCandidate{
		Name:           account.ChannelName,
		ProviderKey:    account.ProviderKind,
		URL:            account.BaseURL,
		APIKey:         account.APIKey,
		ResolvedPrefix: account.Prefix,
		AuthID:         account.AuthID,
	}
}

func directAccountIdentityHeaders(account directAccount, identity clientIdentityContext, settings PluginSettings, payload InterceptRequestPayload) map[string][]string {
	if !account.IdentitySigningEnabled {
		return map[string][]string{}
	}
	candidate := directAccountProviderCandidate(account)
	if account.ProviderKind == directProviderGPT {
		return any2APIIdentityHeaders(identity, settings, candidate)
	}
	spec := providerSpec{
		Name:    account.ChannelName,
		Prefix:  account.Prefix,
		BaseURL: account.BaseURL,
		Models:  directAccountModelSpecs(account),
	}
	return agyIdentityHeaders(identity, settings, candidate, payload, spec)
}

func agyIdentityHeaders(identity clientIdentityContext, settings PluginSettings, candidate providerCandidate, payload InterceptRequestPayload, spec providerSpec) map[string][]string {
	identity = ensureIdentityTimestamp(identity)
	headers := map[string][]string{
		"X-AGY-Principal":         {identity.Principal},
		"X-AGY-Timestamp":         {identity.Timestamp},
		"X-AGY-Client-App":        {identity.ClientApp},
		"X-AGY-Plugin-Version":    {pluginVersion},
		"X-AGY-CPA-Provider-Name": {candidate.Name},
	}
	if identity.ClientInstance != "" {
		headers["X-AGY-Client-Instance"] = []string{identity.ClientInstance}
	}
	if identity.CapabilityProfile != "" {
		headers["X-AGY-Capability-Profile"] = []string{identity.CapabilityProfile}
	}
	if identity.ConnectorID != "" {
		headers["X-AGY-Connector-Id"] = []string{identity.ConnectorID}
	}
	if secret := hmacSecretForCandidate(settings, candidate); secret != "" {
		headers["X-AGY-Signature"] = []string{computeHMAC(identitySignatureMessage(identity, "POST",
			signedUpstreamPath(payload.Model, payload.ToFormat, "", requestPathFromMetadata(payload.Metadata), spec)), secret)}
	}
	return headers
}

func directAccountModelSpecs(account directAccount) []modelSpec {
	models := make([]modelSpec, 0, len(account.Models))
	for _, model := range account.Models {
		if strings.TrimSpace(model.UpstreamID) == "" {
			continue
		}
		models = append(models, modelSpec{
			Name:             model.UpstreamID,
			Alias:            model.Alias,
			Image:            model.Image,
			InputModalities:  append([]string(nil), model.InputModalities...),
			OutputModalities: append([]string(nil), model.OutputModalities...),
			Thinking:         model.Thinking,
		})
	}
	return models
}

func directChannelPayload(account directAccount) map[string]any {
	account = normalizeDirectAccount(account)
	payload := map[string]any{
		"name":     account.ChannelName,
		"prefix":   account.Prefix,
		"base-url": account.BaseURL,
		"priority": account.Priority,
		"disabled": !account.Enabled,
	}
	if account.APIKey != "" {
		payload["api-key-entries"] = []any{map[string]any{"api-key": account.APIKey}}
	}
	if len(account.StaticHeaders) > 0 {
		headers := map[string]any{}
		for key, value := range account.StaticHeaders {
			headers[key] = value
		}
		payload["headers"] = headers
	}
	models := make([]any, 0, len(account.Models))
	for _, model := range account.Models {
		model = normalizeDirectAccountModel(model)
		if !model.Enabled || model.UpstreamID == "" {
			continue
		}
		row := map[string]any{"name": model.UpstreamID}
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
		models = append(models, row)
	}
	payload["models"] = models
	return payload
}

func parseDirectModelCatalog(raw []byte, existing []directAccountModel, limit int) []directAccountModel {
	if limit <= 0 || limit > maxDirectModels {
		limit = maxDirectModels
	}
	ids := directCatalogIDs(raw)
	sortStrings(ids)
	ids = uniqueStrings(ids)
	if len(ids) > limit {
		ids = ids[:limit]
	}
	existingByID := map[string]directAccountModel{}
	for _, model := range normalizeDirectAccountModels(existing) {
		existingByID[strings.ToLower(model.UpstreamID)] = model
	}
	out := make([]directAccountModel, 0, minInt(limit, len(ids)+len(existingByID)))
	seen := map[string]struct{}{}
	for _, id := range ids {
		key := strings.ToLower(id)
		model, exists := existingByID[key]
		if exists {
			model.Unavailable = false
		} else {
			model = directAccountModel{UpstreamID: id, Enabled: true}
		}
		out = append(out, normalizeDirectAccountModel(model))
		seen[key] = struct{}{}
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

func directCatalogIDs(raw []byte) []string {
	var value any
	if errUnmarshal := json.Unmarshal(raw, &value); errUnmarshal != nil {
		return nil
	}
	if root := asMap(value); root != nil {
		if data, ok := mapValue(root, "data"); ok {
			return idsFromCatalogItems(asSlice(data))
		}
	}
	return idsFromCatalogItems(asSlice(value))
}

func idsFromCatalogItems(items []any) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		switch typed := item.(type) {
		case string:
			if id := strings.TrimSpace(typed); id != "" {
				ids = append(ids, id)
			}
		default:
			if itemMap := asMap(typed); itemMap != nil {
				if id, ok := stringValue(itemMap, "id", "name"); ok && id != "" {
					ids = append(ids, id)
				}
			}
		}
	}
	return ids
}
