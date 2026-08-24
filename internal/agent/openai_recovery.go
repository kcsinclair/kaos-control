// SPDX-License-Identifier: AGPL-3.0-or-later

package agent

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

var (
	// xmlFuncRegex matches <function=NAME>...</function> or <function name="NAME">...</function>
	xmlFuncRegex = regexp.MustCompile(`(?s)<function(?:=|\s+name=")([^>"]+)"?>(.*?)</function>`)
	// xmlParamRegex matches <parameter=KEY>VALUE</parameter> or <parameter name="KEY">VALUE</parameter>
	xmlParamRegex = regexp.MustCompile(`(?s)<parameter(?:=|\s+name=")([^>"]+)"?>(.*?)</parameter>`)
	// jsonToolCallRegex matches <tool_call>JSON</tool_call>
	jsonToolCallRegex = regexp.MustCompile(`(?s)<tool_call>(.*?)</tool_call>`)
)

// ParseNativeCalls attempts to recover tool calls from content emitted in native syntax
// (<function=...> or <tool_call>). Returns the recovered ToolCalls and the cleaned remaining content.
func ParseNativeCalls(content string) ([]ToolCall, string) {
	if content == "" {
		return nil, ""
	}

	var recovered []ToolCall
	callIdx := 0

	// 1. Check for JSON <tool_call>...</tool_call>
	matchesJSON := jsonToolCallRegex.FindAllStringSubmatchIndex(content, -1)
	if len(matchesJSON) > 0 {
		var cleanBuf strings.Builder
		lastEnd := 0
		for _, idxs := range matchesJSON {
			startTag := idxs[0]
			endTag := idxs[1]
			bodyStart := idxs[2]
			bodyEnd := idxs[3]

			cleanBuf.WriteString(content[lastEnd:startTag])
			lastEnd = endTag

			rawJSON := strings.TrimSpace(content[bodyStart:bodyEnd])
			var parsed map[string]any
			if err := json.Unmarshal([]byte(rawJSON), &parsed); err == nil {
				name, _ := parsed["name"].(string)
				if name == "" {
					// Check "function" object if present
					if fnObj, ok := parsed["function"].(map[string]any); ok {
						name, _ = fnObj["name"].(string)
						if args, ok := fnObj["arguments"]; ok {
							parsed["arguments"] = args
						}
					}
				}
				if name != "" {
					var argsStr string
					if rawArgs, ok := parsed["arguments"]; ok {
						if s, ok := rawArgs.(string); ok {
							argsStr = s
						} else if b, err := json.Marshal(rawArgs); err == nil {
							argsStr = string(b)
						}
					} else if rawParams, ok := parsed["parameters"]; ok {
						if s, ok := rawParams.(string); ok {
							argsStr = s
						} else if b, err := json.Marshal(rawParams); err == nil {
							argsStr = string(b)
						}
					}
					if argsStr == "" {
						argsStr = "{}"
					}
					callIdx++
					recovered = append(recovered, ToolCall{
						ID:   fmt.Sprintf("call_recov_%d", callIdx),
						Type: "function",
						Function: FunctionCallInfo{
							Name:      name,
							Arguments: argsStr,
						},
					})
				}
			}
		}
		cleanBuf.WriteString(content[lastEnd:])
		remaining := strings.TrimSpace(cleanBuf.String())
		if len(recovered) > 0 {
			return recovered, remaining
		}
	}

	// 2. Check for XML <function=NAME>...</function>
	matchesXML := xmlFuncRegex.FindAllStringSubmatchIndex(content, -1)
	if len(matchesXML) > 0 {
		var cleanBuf strings.Builder
		lastEnd := 0
		for _, idxs := range matchesXML {
			startTag := idxs[0]
			endTag := idxs[1]
			nameStart := idxs[2]
			nameEnd := idxs[3]
			bodyStart := idxs[4]
			bodyEnd := idxs[5]

			cleanBuf.WriteString(content[lastEnd:startTag])
			lastEnd = endTag

			fnName := strings.TrimSpace(content[nameStart:nameEnd])
			fnBody := content[bodyStart:bodyEnd]

			paramMatches := xmlParamRegex.FindAllStringSubmatch(fnBody, -1)
			argsMap := make(map[string]any)
			for _, pm := range paramMatches {
				pKey := strings.TrimSpace(pm[1])
				pVal := strings.TrimSpace(pm[2])
				// Try unmarshaling if JSON (e.g. numbers, objects), otherwise keep as string
				var jsonVal any
				if err := json.Unmarshal([]byte(pVal), &jsonVal); err == nil && (strings.HasPrefix(pVal, "{") || strings.HasPrefix(pVal, "[") || pVal == "true" || pVal == "false" || pVal == "null") {
					argsMap[pKey] = jsonVal
				} else {
					argsMap[pKey] = pVal
				}
			}

			argsBytes, err := json.Marshal(argsMap)
			argsStr := "{}"
			if err == nil {
				argsStr = string(argsBytes)
			}

			callIdx++
			recovered = append(recovered, ToolCall{
				ID:   fmt.Sprintf("call_recov_%d", callIdx),
				Type: "function",
				Function: FunctionCallInfo{
					Name:      fnName,
					Arguments: argsStr,
				},
			})
		}
		cleanBuf.WriteString(content[lastEnd:])
		remaining := strings.TrimSpace(cleanBuf.String())
		if len(recovered) > 0 {
			return recovered, remaining
		}
	}

	return nil, content
}
