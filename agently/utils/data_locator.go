package utils

import (
	"encoding/json"
	"strings"
)

func LocatePathInData(data any, path string, style string, defaultValue any) any {
	if strings.TrimSpace(path) == "" {
		if data == nil {
			return defaultValue
		}
		return data
	}
	if style == "" {
		style = "dot"
	}
	value := GetValueByPath(data, path, style)
	if value == nil {
		return defaultValue
	}
	return value
}

func LocateAllJSON(originalText string) []string {
	jsonBlocks := make([]string, 0)
	stage := 1
	blockNum := 0
	layer := 0
	skipNext := false
	inQuote := false

	for i := 0; i < len(originalText); i++ {
		char := originalText[i]
		if skipNext {
			skipNext = false
			continue
		}
		if stage == 1 {
			if char == '\\' {
				skipNext = true
				continue
			}
			if char == '[' || char == '{' {
				jsonBlocks = append(jsonBlocks, string(char))
				stage = 2
				layer = 1
			}
			continue
		}

		if !inQuote {
			if char == '\\' {
				skipNext = true
				if i+1 < len(originalText) && originalText[i+1] == '"' {
					char = '"'
				} else {
					continue
				}
			}
			if char == '"' {
				inQuote = true
			}
			if char == '[' || char == '{' {
				layer++
			} else if char == ']' || char == '}' {
				layer--
			}
			jsonBlocks[blockNum] += string(char)
		} else {
			if char == '\\' && i+1 < len(originalText) {
				jsonBlocks[blockNum] += string(char) + string(originalText[i+1])
				skipNext = true
				continue
			}
			if char == '\n' {
				jsonBlocks[blockNum] += "\\n"
				continue
			}
			if char == '\t' {
				jsonBlocks[blockNum] += "\\t"
				continue
			}
			if char == '"' {
				inQuote = false
			}
			jsonBlocks[blockNum] += string(char)
		}

		if layer == 0 {
			blockNum++
			stage = 1
		}
	}
	return jsonBlocks
}

func LocateOutputJSON(originalText string, outputSchema map[string]any) string {
	all := LocateAllJSON(originalText)
	if len(all) == 0 {
		return ""
	}
	if len(all) == 1 {
		return all[0]
	}

	for i := 0; i < len(all)-1; i++ {
		m := map[string]any{}
		if err := json.Unmarshal([]byte(all[i]), &m); err != nil {
			continue
		}
		for key := range m {
			if _, ok := outputSchema[key]; ok {
				return all[i]
			}
		}
	}
	return all[len(all)-1]
}
