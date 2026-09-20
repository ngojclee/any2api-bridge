package main

import (
	_ "embed"
	"net/http"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

// iconBytes is the plugin mark cached in the repository so the management panel can
// resolve it without depending on GitHub raw availability at page load time.
//
//go:embed assets/logo.svg
var iconBytes []byte

const (
	iconRoutePath   = "/icon"
	iconContentType = "image/svg+xml"
)

func iconResponse() pluginapi.ManagementResponse {
	return pluginapi.ManagementResponse{
		StatusCode: http.StatusOK,
		Headers: http.Header{
			"Content-Type":  []string{iconContentType},
			"Cache-Control": []string{"public, max-age=86400"},
		},
		Body: iconBytes,
	}
}
