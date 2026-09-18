package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDirectAccountValidationRejectsUnsafeAndConflictingAccounts(t *testing.T) {
	validAGY := directAccount{
		AccountID:              "agy-prod",
		ProviderKind:           directProviderAGY,
		ChannelName:            "AGY Direct",
		Prefix:                 "agy",
		Enabled:                true,
		IdentitySigningEnabled: true,
	}
	validGPT := directAccount{
		AccountID:              "gpt-prod",
		ProviderKind:           directProviderGPT,
		ChannelName:            "GPT Direct",
		Prefix:                 "gpt",
		Enabled:                true,
		IdentitySigningEnabled: true,
	}
	if err := validateDirectAccounts([]directAccount{validAGY, validGPT}); err != nil {
		t.Fatalf("valid direct accounts rejected: %v", err)
	}

	cases := []struct {
		name     string
		accounts []directAccount
		want     string
	}{
		{
			name:     "invalid account id",
			accounts: []directAccount{{AccountID: "bad id", ProviderKind: directProviderAGY, ChannelName: "A", Prefix: "a", Enabled: true}},
			want:     "invalid account_id",
		},
		{
			name: "duplicate account id",
			accounts: []directAccount{
				{AccountID: "same", ProviderKind: directProviderAGY, ChannelName: "A", Prefix: "a", Enabled: true},
				{AccountID: "same", ProviderKind: directProviderGPT, ChannelName: "B", Prefix: "b", Enabled: true},
			},
			want: "duplicates account_id",
		},
		{
			name:     "unsafe prefix",
			accounts: []directAccount{{AccountID: "a", ProviderKind: directProviderAGY, ChannelName: "A", Prefix: "bad/prefix", Enabled: true}},
			want:     "unsafe prefix",
		},
		{
			name: "duplicate active prefix",
			accounts: []directAccount{
				{AccountID: "a", ProviderKind: directProviderAGY, ChannelName: "A", Prefix: "same", Enabled: true},
				{AccountID: "b", ProviderKind: directProviderGPT, ChannelName: "B", Prefix: "same", Enabled: true},
			},
			want: "duplicates active prefix",
		},
		{
			name: "cross kind channel collision",
			accounts: []directAccount{
				{AccountID: "a", ProviderKind: directProviderAGY, ChannelName: "Shared", Prefix: "a", Enabled: true},
				{AccountID: "b", ProviderKind: directProviderGPT, ChannelName: "Shared", Prefix: "b", Enabled: true},
			},
			want: "across provider kinds",
		},
		{
			name: "duplicate active model alias",
			accounts: []directAccount{
				{AccountID: "a", ProviderKind: directProviderAGY, ChannelName: "A", Prefix: "a", Enabled: true, Models: []directAccountModel{{UpstreamID: "upstream-a", Alias: "shared", Enabled: true}}},
				{AccountID: "b", ProviderKind: directProviderGPT, ChannelName: "B", Prefix: "b", Enabled: true, Models: []directAccountModel{{UpstreamID: "upstream-b", Alias: "shared", Enabled: true}}},
			},
			want: "model alias",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateDirectAccounts(tc.accounts)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("validation error = %v, want containing %q", err, tc.want)
			}
		})
	}
}

func TestDirectAccountReadViewRedactsSecrets(t *testing.T) {
	account := directAccount{
		AccountID:              "gpt-prod",
		ProviderKind:           directProviderGPT,
		ChannelName:            "GPT Direct",
		Prefix:                 "gpt",
		BaseURL:                "https://gpt2api.example/v1?token=hidden",
		Enabled:                true,
		Priority:               50,
		IdentitySigningEnabled: true,
		APIKey:                 "secret-api-key",
		StaticHeaders: map[string]string{
			"X-Static-Secret": "secret-header",
		},
		Models: []directAccountModel{
			{UpstreamID: "gpt-5.6", Alias: "chatgpt-main", Enabled: true},
			{UpstreamID: "hidden", Alias: "off", Enabled: false},
		},
	}
	channel := directChannelPayload(account)
	rawChannel, _ := json.Marshal(channel)
	if !strings.Contains(string(rawChannel), "secret-api-key") || !strings.Contains(string(rawChannel), "secret-header") {
		t.Fatalf("write payload did not include write-only values: %s", rawChannel)
	}
	models := channel["models"].([]any)
	if len(models) != 1 {
		t.Fatalf("channel should publish only enabled models, got %#v", models)
	}
	first := models[0].(map[string]any)
	if first["name"] != "gpt-5.6" || first["alias"] != "chatgpt-main" {
		t.Fatalf("model alias was not preserved: %#v", first)
	}

	view := directAccountReadView(account)
	rawView, _ := json.Marshal(view)
	for _, secret := range []string{"secret-api-key", "secret-header", "?token=hidden"} {
		if strings.Contains(string(rawView), secret) {
			t.Fatalf("read view leaked %q: %s", secret, rawView)
		}
	}
	if !view.APIKeyConfigured || len(view.Headers) != 1 || !view.Headers[0].Configured {
		t.Fatalf("redacted configured state missing: %+v", view)
	}
	if len(view.Models) != 2 || view.Models[0].Alias != "chatgpt-main" {
		t.Fatalf("read view lost model state: %+v", view.Models)
	}
}

func TestDirectManagementStateIsRedacted(t *testing.T) {
	settings := defaultPluginSettings()
	settings.DirectModeEnabled = true
	settings.DirectAccounts = []directAccount{{
		AccountID:              "agy-prod",
		ProviderKind:           directProviderAGY,
		ChannelName:            "AGY Direct",
		Prefix:                 "agy",
		Enabled:                true,
		IdentitySigningEnabled: true,
		APIKey:                 "secret-api-key",
		StaticHeaders:          map[string]string{"X-Secret": "secret-header"},
	}}
	state := directAccountsManagementState(settings)
	raw, _ := json.Marshal(state)
	if !state.DirectModeEnabled || state.AccountCount != 1 {
		t.Fatalf("direct state = %+v", state)
	}
	for _, secret := range []string{"secret-api-key", "secret-header"} {
		if strings.Contains(string(raw), secret) {
			t.Fatalf("management state leaked %q: %s", secret, raw)
		}
	}
}
