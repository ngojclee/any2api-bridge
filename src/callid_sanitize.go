package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
)

const maxOpenAICallIDLength = 64

func shouldNormalizeOpenAICallIDs(payload InterceptRequestPayload) bool {
	toFormat := strings.ToLower(strings.TrimSpace(payload.ToFormat))
	switch toFormat {
	case "codex", "openai", "openai-response", "openai-responses":
		return true
	}
	if payload.Metadata != nil {
		authID := strings.ToLower(strings.TrimSpace(metadataString(
			payload.Metadata,
			"selected_auth_id", "selected-auth-id", "auth_id", "auth-id",
		)))
		if strings.HasPrefix(authID, "codex:") {
			return true
		}
		if compatAuthChannelName(authID) != "" && strings.EqualFold(compatAuthChannelName(authID), "chatgpt") {
			return true
		}
	}
	return false
}

func mapOversizedOpenAICallID(raw string) string {
	if len(raw) <= maxOpenAICallIDLength {
		return raw
	}
	sum := sha256.Sum256([]byte(raw))
	return "call_" + hex.EncodeToString(sum[:])[:48]
}

func mapOversizedOpenAIItemID(raw string) string {
	if len(raw) <= maxOpenAICallIDLength && !strings.HasPrefix(raw, "cpa_tid_") {
		return raw
	}
	sum := sha256.Sum256([]byte(raw))
	return "fc_" + hex.EncodeToString(sum[:])[:48]
}

// hasPotentialOversizedCallID performs a zero-allocation byte scan before any
// JSON decode so normal traffic pays virtually zero cost.
func hasPotentialOversizedCallID(body []byte) bool {
	if len(body) == 0 {
		return false
	}
	if bytes.Contains(body, []byte("cpa_tid_")) {
		return true
	}
	for _, needle := range [][]byte{[]byte(`"call_id"`), []byte(`"tool_call_id"`)} {
		search := body
		for {
			idx := bytes.Index(search, needle)
			if idx < 0 {
				break
			}
			rest := search[idx+len(needle):]
			search = rest
			// Advance past whitespace and ':' to the opening quote.
			i := 0
			for i < len(rest) && (rest[i] == ' ' || rest[i] == '\t' || rest[i] == '\r' || rest[i] == '\n') {
				i++
			}
			if i >= len(rest) || rest[i] != ':' {
				continue
			}
			i++
			for i < len(rest) && (rest[i] == ' ' || rest[i] == '\t' || rest[i] == '\r' || rest[i] == '\n') {
				i++
			}
			if i >= len(rest) || rest[i] != '"' {
				continue
			}
			i++
			start := i
			for i < len(rest) {
				if rest[i] == '\\' {
					i += 2
					continue
				}
				if rest[i] == '"' {
					break
				}
				i++
			}
			if i-start > maxOpenAICallIDLength {
				return true
			}
		}
	}
	return false
}

func normalizeOpenAIRequestCallIDs(body []byte) ([]byte, bool) {
	if !hasPotentialOversizedCallID(body) {
		return nil, false
	}
	var root map[string]any
	if errUnmarshal := json.Unmarshal(body, &root); errUnmarshal != nil || root == nil {
		return nil, false
	}
	changed := false
	if rawInput, ok := root["input"]; ok {
		if items, isSlice := rawInput.([]any); isSlice {
			for _, elem := range items {
				if sanitizeResponsesInputItem(elem) {
					changed = true
				}
			}
		}
	}
	if rawMessages, ok := root["messages"]; ok {
		if msgs, isSlice := rawMessages.([]any); isSlice {
			for _, elem := range msgs {
				if sanitizeChatMessageCallIDs(elem) {
					changed = true
				}
			}
		}
	}
	if !changed {
		return nil, false
	}
	encoded, errMarshal := json.Marshal(root)
	if errMarshal != nil {
		return nil, false
	}
	return encoded, true
}

func sanitizeResponsesInputItem(node any) bool {
	item, ok := node.(map[string]any)
	if !ok || item == nil {
		return false
	}
	changed := false
	if callID, ok := item["call_id"].(string); ok && len(callID) > maxOpenAICallIDLength {
		item["call_id"] = mapOversizedOpenAICallID(callID)
		changed = true
	}
	if idVal, ok := item["id"].(string); ok {
		// OpenAI Responses API rejects synthetic CPA item IDs ("cpa_tid_v1_*")
		// and oversized item IDs (> 64 chars). Dropping non-OpenAI item IDs
		// matches codex-rs ModelClient::prepare_response_items_for_request.
		if strings.HasPrefix(idVal, "cpa_tid_") || strings.HasPrefix(idVal, "call_") || len(idVal) > maxOpenAICallIDLength {
			itemType, _ := item["type"].(string)
			if itemType == "item_reference" {
				item["id"] = mapOversizedOpenAIItemID(idVal)
			} else {
				delete(item, "id")
			}
			changed = true
		}
	}
	if itemID, ok := item["item_id"].(string); ok && (strings.HasPrefix(itemID, "cpa_tid_") || len(itemID) > maxOpenAICallIDLength) {
		item["item_id"] = mapOversizedOpenAIItemID(itemID)
		changed = true
	}
	return changed
}

func sanitizeChatMessageCallIDs(node any) bool {
	msg, ok := node.(map[string]any)
	if !ok || msg == nil {
		return false
	}
	changed := false
	if toolCallID, ok := msg["tool_call_id"].(string); ok && len(toolCallID) > maxOpenAICallIDLength {
		msg["tool_call_id"] = mapOversizedOpenAICallID(toolCallID)
		changed = true
	}
	if rawCalls, ok := msg["tool_calls"].([]any); ok {
		for _, elem := range rawCalls {
			callMap, isMap := elem.(map[string]any)
			if !isMap || callMap == nil {
				continue
			}
			if idVal, ok := callMap["id"].(string); ok && len(idVal) > maxOpenAICallIDLength {
				callMap["id"] = mapOversizedOpenAICallID(idVal)
				changed = true
			}
		}
	}
	return changed
}
