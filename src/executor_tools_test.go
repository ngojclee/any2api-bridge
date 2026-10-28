package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"reflect"
	"testing"
)

// Client-executed tools travel through the bridge as ordinary JSON fields.
// Both executor rewrites (rewritePayloadModelWithEffort and
// stripPayloadModelPrefix) round-trip the body through map[string]any, which
// is the only place unknown fields could be dropped before the upstream call.
// This pins every tool contract field through the full normalize + build path.
func TestExecutorPreservesClientToolContractFields(t *testing.T) {
	loadMirror(t)
	root, errParse := parseYAMLMap([]byte(mirrorConfigYAML))
	if errParse != nil {
		t.Fatal(errParse)
	}
	spec, found := extractProviderSpec(root, currentPluginSettings())
	if !found {
		t.Fatal("no mirrored provider")
	}

	payload := []byte(`{
		"model": "agy/gemini-3.1-pro",
		"stream": false,
		"tools": [{
			"type": "function",
			"function": {
				"name": "get_weather",
				"description": "Current weather for a city",
				"parameters": {
					"type": "object",
					"properties": {"city": {"type": "string"}},
					"required": ["city"]
				}
			}
		}],
		"tool_choice": {"type": "function", "function": {"name": "get_weather"}},
		"parallel_tool_calls": false,
		"agy_tool_execution": "client",
		"messages": [
			{"role": "user", "content": "weather in Hanoi?"},
			{
				"role": "assistant",
				"content": null,
				"tool_calls": [{
					"id": "call_abc123",
					"type": "function",
					"function": {"name": "get_weather", "arguments": "{\"city\":\"Hanoi\"}"}
				}]
			},
			{"role": "tool", "tool_call_id": "call_abc123", "content": "25C sunny"}
		]
	}`)

	req := executorRequest{
		Model:   "agy/gemini-3.1-pro",
		Format:  "openai",
		Payload: payload,
	}
	normalizeExecutorModel(&req, spec)
	built, errBuild := buildUpstreamRequest(req, spec, identityFromExecutorRequest(req))
	if errBuild != nil {
		t.Fatal(errBuild)
	}

	var want, got map[string]any
	if errUnmarshal := json.Unmarshal(payload, &want); errUnmarshal != nil {
		t.Fatal(errUnmarshal)
	}
	// The only sanctioned rewrite is the model prefix strip.
	want["model"] = "gemini-3.1-pro"
	if errUnmarshal := json.Unmarshal(unb64(built.Body), &got); errUnmarshal != nil {
		t.Fatalf("upstream body is not JSON: %v", errUnmarshal)
	}
	if !reflect.DeepEqual(got, want) {
		gotRaw, _ := json.Marshal(got)
		t.Fatalf("upstream body dropped or altered tool fields:\n got: %s\nwant keys preserved: tools, tool_choice, parallel_tool_calls, agy_tool_execution, messages[].tool_calls, messages[].tool_call_id", gotRaw)
	}
}

// A tool_choice variant round: "auto" is a string, not an object, and must
// survive verbatim alongside a second tool result pair.
func TestExecutorPreservesToolChoiceVariants(t *testing.T) {
	loadMirror(t)
	root, _ := parseYAMLMap([]byte(mirrorConfigYAML))
	spec, found := extractProviderSpec(root, currentPluginSettings())
	if !found {
		t.Fatal("no mirrored provider")
	}
	for _, choice := range []string{`"auto"`, `"required"`, `"none"`} {
		payload := []byte(`{"model":"agy/gemini-3.1-pro","tool_choice":` + choice + `,"parallel_tool_calls":true,"messages":[]}`)
		req := executorRequest{Model: "agy/gemini-3.1-pro", Payload: payload}
		normalizeExecutorModel(&req, spec)
		built, errBuild := buildUpstreamRequest(req, spec, identityFromExecutorRequest(req))
		if errBuild != nil {
			t.Fatal(errBuild)
		}
		var got map[string]any
		if errUnmarshal := json.Unmarshal(unb64(built.Body), &got); errUnmarshal != nil {
			t.Fatal(errUnmarshal)
		}
		var wantChoice any
		if errUnmarshal := json.Unmarshal([]byte(choice), &wantChoice); errUnmarshal != nil {
			t.Fatal(errUnmarshal)
		}
		if !reflect.DeepEqual(got["tool_choice"], wantChoice) {
			t.Fatalf("tool_choice %s became %#v", choice, got["tool_choice"])
		}
		if got["parallel_tool_calls"] != true {
			t.Fatalf("parallel_tool_calls dropped for tool_choice %s", choice)
		}
	}
}

// The tool-call stream is the load-bearing case: agy2api emits one
// delta.tool_calls chunk (each entry carrying its own index), then a
// finish_reason=tool_calls frame, then [DONE]. The parser must emit the first
// two byte-identical and still drop only [DONE], even when a read splits the
// arguments mid-frame.
func TestSSEStreamParserPreservesToolCallDelta(t *testing.T) {
	toolCallFrame := `{"id":"chatcmpl-1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_abc123","type":"function","function":{"name":"get_weather","arguments":"{\"city\":\"Hanoi\"}"}}]}}]}`
	finishFrame := `{"id":"chatcmpl-1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`

	// Split inside the arguments string so reassembly is exercised too.
	cut := len(`data: {"id":"chatcmpl-1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_abc123","type":"function","function":{"name":"get_weather","arguments":"{\"ci`)
	first := `data: ` + toolCallFrame
	stream := first[:cut] + first[cut:] + "\n\ndata: " + finishFrame + "\n\ndata: [DONE]\n\n"

	parser := &sseStreamParser{}
	var got [][]byte
	got = append(got, parser.feed([]byte(stream[:cut]))...)
	got = append(got, parser.feed([]byte(stream[cut:]))...)
	got = append(got, parser.flush()...)

	if len(got) != 2 {
		t.Fatalf("emitted %d events, want 2 (tool_calls delta + finish): %q", len(got), got)
	}
	if string(got[0]) != toolCallFrame {
		t.Fatalf("tool_calls delta altered:\n got: %s\nwant: %s", got[0], toolCallFrame)
	}
	if string(got[1]) != finishFrame {
		t.Fatalf("finish frame altered:\n got: %s\nwant: %s", got[1], finishFrame)
	}

	// The emit envelope base64-wraps the raw event; decode proves no escaping
	// or filtering touched the payload before host.stream.emit sees it.
	emitRaw, errMarshal := streamEmitPayload("bridge-stream-1", got[0])
	if errMarshal != nil {
		t.Fatal(errMarshal)
	}
	var emitDecoded struct {
		StreamID string `json:"stream_id"`
		Payload  string `json:"payload"`
	}
	if errUnmarshal := json.Unmarshal(emitRaw, &emitDecoded); errUnmarshal != nil {
		t.Fatal(errUnmarshal)
	}
	event, errDecode := base64.StdEncoding.DecodeString(emitDecoded.Payload)
	if errDecode != nil {
		t.Fatal(errDecode)
	}
	if !bytes.Equal(event, got[0]) {
		t.Fatal("emit envelope corrupted the tool_calls event bytes")
	}
}

// An agy2api tool_call_invalid surfaces as an event: error block whose data
// must reach the client untouched so the harness can display it, while the
// terminal [DONE] marker stays swallowed because CPA writes its own tail.
func TestSSEStreamParserPassesErrorEventsAndSkipsDone(t *testing.T) {
	errorPayload := `{"error":{"type":"tool_call_invalid","code":"bad_arguments","message":"arguments failed schema validation"}}`
	stream := "event: error\ndata: " + errorPayload + "\n\ndata: [DONE]\n\n"

	parser := &sseStreamParser{}
	got := parser.feed([]byte(stream))
	got = append(got, parser.flush()...)

	if len(got) != 1 {
		t.Fatalf("emitted %d events, want only the error frame: %q", len(got), got)
	}
	if string(got[0]) != errorPayload {
		t.Fatalf("error event altered:\n got: %s\nwant: %s", got[0], errorPayload)
	}
}
