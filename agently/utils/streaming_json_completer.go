package utils

import "strings"

// StreamingJSONCompleter completes potentially truncated streamed JSON fragments.
type StreamingJSONCompleter struct {
	buffer string
}

func NewStreamingJSONCompleter() *StreamingJSONCompleter {
	return &StreamingJSONCompleter{}
}

func (s *StreamingJSONCompleter) Reset(data string) {
	s.buffer = data
}

func (s *StreamingJSONCompleter) Append(data string) {
	s.buffer += data
}

func (s *StreamingJSONCompleter) Complete() string {
	buf := s.buffer
	stack := make([]rune, 0)
	inString := false
	escape := false
	stringChar := rune(0)
	comment := "" // "", "//", "/*"

	runes := []rune(buf)
	for i := 0; i < len(runes); i++ {
		ch := runes[i]
		next := rune(0)
		if i+1 < len(runes) {
			next = runes[i+1]
		}

		if comment == "" {
			if inString {
				if escape {
					escape = false
					continue
				}
				if ch == '\\' {
					escape = true
					continue
				}
				if ch == stringChar {
					inString = false
					stringChar = 0
				}
				continue
			}

			if ch == '"' || ch == '\'' {
				inString = true
				stringChar = ch
				continue
			}
			if ch == '/' && next == '/' {
				comment = "//"
				i++
				continue
			}
			if ch == '/' && next == '*' {
				comment = "/*"
				i++
				continue
			}
			if ch == '{' || ch == '[' {
				stack = append(stack, ch)
				continue
			}
			if ch == '}' || ch == ']' {
				if len(stack) == 0 {
					continue
				}
				top := stack[len(stack)-1]
				if (top == '{' && ch == '}') || (top == '[' && ch == ']') {
					stack = stack[:len(stack)-1]
				}
			}
			continue
		}

		if comment == "//" {
			if ch == '\n' || ch == '\r' {
				comment = ""
			}
			continue
		}
		if comment == "/*" && ch == '*' && next == '/' {
			comment = ""
			i++
		}
	}

	if inString && stringChar != 0 {
		buf += string(stringChar)
	}
	if comment == "//" {
		buf += "\n"
	} else if comment == "/*" {
		buf += "*/"
	}

	if len(stack) > 0 {
		closings := strings.Builder{}
		for i := len(stack) - 1; i >= 0; i-- {
			if stack[i] == '{' {
				closings.WriteRune('}')
			} else if stack[i] == '[' {
				closings.WriteRune(']')
			}
		}
		buf += closings.String()
	}
	return buf
}
