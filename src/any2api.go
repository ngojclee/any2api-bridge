package main

import "strings"

const (
	any2APIHeaderPrincipal      = "X-Any2API-Principal"
	any2APIHeaderClientApp      = "X-Any2API-Client-App"
	any2APIHeaderClientInstance = "X-Any2API-Client-Instance"
	any2APIHeaderConversationID = "X-Any2API-Conversation-Id"
	any2APIHeaderTimestamp      = "X-Any2API-Identity-Timestamp"
	any2APIHeaderSignature      = "X-Any2API-Identity-Signature"
)

type identityHeaderProfile string

const (
	identityHeaderProfileAGY     identityHeaderProfile = "agy2api"
	identityHeaderProfileAny2API identityHeaderProfile = "any2api"
)

func identityHeaderProfileForCandidate(candidate providerCandidate) identityHeaderProfile {
	if isGPT2APITarget(candidate.Name, candidate.ProviderKey, candidate.URL, candidate.ResolvedPrefix) {
		return identityHeaderProfileAny2API
	}
	return identityHeaderProfileAGY
}

func identityHeaderProfileForSpec(spec providerSpec) identityHeaderProfile {
	if isGPT2APITarget(spec.Name, spec.Prefix, spec.BaseURL) {
		return identityHeaderProfileAny2API
	}
	return identityHeaderProfileAGY
}

func isGPT2APITarget(values ...string) bool {
	for _, value := range values {
		if strings.Contains(strings.ToLower(strings.TrimSpace(value)), "gpt2api") {
			return true
		}
	}
	return false
}

func any2APIIdentityHeaders(identity clientIdentityContext, settings PluginSettings, candidate providerCandidate) map[string][]string {
	identity = ensureIdentityTimestamp(identity)
	headers := map[string][]string{}
	setIf := func(key, value string) {
		if strings.TrimSpace(value) != "" {
			headers[key] = []string{value}
		}
	}
	setIf(any2APIHeaderPrincipal, identity.Principal)
	setIf(any2APIHeaderClientApp, identity.ClientApp)
	setIf(any2APIHeaderClientInstance, identity.ClientInstance)
	setIf(any2APIHeaderConversationID, identity.SessionID)
	setIf(any2APIHeaderTimestamp, identity.Timestamp)
	if secret := hmacSecretForCandidate(settings, candidate); secret != "" && identity.Principal != "" {
		headers[any2APIHeaderSignature] = []string{computeHMAC(any2APIIdentitySignatureMessage(identity), secret)}
	}
	return headers
}
