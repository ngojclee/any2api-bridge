package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

func TestIconRouteServesThePluginMark(t *testing.T) {
	request, errMarshal := json.Marshal(pluginapi.ManagementRequest{
		Method: http.MethodGet,
		Path:   "/v0/resource" + managementBasePath + iconRoutePath,
	})
	if errMarshal != nil {
		t.Fatal(errMarshal)
	}
	raw, errHandle := handleManagement(request)
	if errHandle != nil {
		t.Fatalf("icon route: %v", errHandle)
	}
	var envelope struct {
		Result struct {
			StatusCode int         `json:"StatusCode"`
			Headers    http.Header `json:"headers"`
			Body       []byte      `json:"body"`
		} `json:"result"`
	}
	if errUnmarshal := json.Unmarshal(raw, &envelope); errUnmarshal != nil {
		t.Fatal(errUnmarshal)
	}
	if envelope.Result.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", envelope.Result.StatusCode)
	}
	if got := envelope.Result.Headers.Get("Content-Type"); got != iconContentType {
		t.Fatalf("content type = %q, want %q", got, iconContentType)
	}
	if len(envelope.Result.Body) < 100 {
		t.Fatalf("icon body is %d bytes, want the real mark", len(envelope.Result.Body))
	}
}

func TestIconRouteIsRegisteredAsAResource(t *testing.T) {
	raw := string(handleManagementRegister())
	if !strings.Contains(raw, `"Path":"`+iconRoutePath+`"`) {
		t.Fatalf("resource route %q missing from registration", iconRoutePath)
	}
	if !strings.HasSuffix(pluginRegistration().Metadata.Logo, iconRoutePath) {
		t.Fatalf("logo = %q, want same-origin icon route", pluginRegistration().Metadata.Logo)
	}
}
