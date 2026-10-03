package main

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// Regression: thinkingSpec was embedded into CPA config maps without yaml
// tags, so yaml.v3 lowercased the Go field names and produced
// "zeroallowed"/"dynamicallowed". CPA's strict ThinkingSupport decoder rejects
// those keys and every management-API save of config.yaml failed with
// "field dynamicallowed not found in type registry.ThinkingSupport". The
// schema keys are kebab-case.
func TestThinkingSpecMarshalsCPASchemaKeys(t *testing.T) {
	model := directAccountModel{
		UpstreamID: "gemini-3.8-flash",
		Enabled:    true,
		Thinking: &thinkingSpec{
			Min:            1024,
			Max:            32768,
			ZeroAllowed:    true,
			DynamicAllowed: true,
			Levels:         []string{"none", "low", "medium", "high"},
		},
	}
	account := directAccount{
		AccountID:   "agy-prod",
		ChannelName: "Antigravity",
		Prefix:      "agy",
		Enabled:     true,
		Models:      []directAccountModel{model},
	}
	for name, section := range map[string]any{
		"store":    directAccountsForConfig([]directAccount{account})[0],
		"provider": directChannelPayload(account)["models"],
	} {
		out, errMarshal := yaml.Marshal(section)
		if errMarshal != nil {
			t.Fatalf("%s marshal: %v", name, errMarshal)
		}
		text := string(out)
		for _, bad := range []string{"zeroallowed", "dynamicallowed", "inputmodalities", "outputmodalities"} {
			if strings.Contains(text, bad+":") {
				t.Fatalf("%s emitted non-schema key %q:\n%s", name, bad, text)
			}
		}
		for _, want := range []string{"zero-allowed: true", "dynamic-allowed: true", "min: 1024", "max: 32768"} {
			if !strings.Contains(text, want) {
				t.Fatalf("%s missing %q:\n%s", name, want, text)
			}
		}
	}
}

// Configs written by the broken version carry "zeroallowed"/"dynamicallowed";
// readers must keep accepting them so a stored account does not lose its
// thinking metadata before the next write normalizes the keys.
func TestThinkingSpecReadsLegacyUntaggedKeys(t *testing.T) {
	raw := map[string]any{
		"upstream_id": "m",
		"thinking": map[string]any{
			"zeroallowed":    true,
			"dynamicallowed": true,
			"levels":         []any{"low", "high"},
		},
	}
	model := directAccountModelFromMap(raw)
	if model.Thinking == nil || !model.Thinking.ZeroAllowed || !model.Thinking.DynamicAllowed {
		t.Fatalf("account-model thinking did not parse legacy keys: %+v", model.Thinking)
	}
	provider := map[string]any{
		"name": "p",
		"models": []any{map[string]any{
			"name": "m",
			"thinking": map[string]any{
				"zeroallowed":    true,
				"dynamicallowed": true,
			},
		}},
	}
	specs := compatModels(provider)
	if len(specs) != 1 || specs[0].Thinking == nil || !specs[0].Thinking.ZeroAllowed || !specs[0].Thinking.DynamicAllowed {
		t.Fatalf("provider thinking did not parse legacy keys: %+v", specs)
	}
}

func TestProbeModelsURLAppendsVariantQuery(t *testing.T) {
	spec := providerSpec{BaseURL: "https://agy2api.example/v1/"}
	if got := probeModelsURL(spec, ""); got != "https://agy2api.example/v1/models" {
		t.Fatalf("bare url = %q", got)
	}
	if got := probeModelsURL(spec, "include_variants=1"); got != "https://agy2api.example/v1/models?include_variants=1" {
		t.Fatalf("variant url = %q", got)
	}
	// The signed upstream path is the bare path — the query must not leak in.
}
