package partialjson

import (
	"bytes"
	"unicode/utf8"
)

// Parser handles incomplete JSON parsing.
type Parser struct {
	strict bool // If true, reject newlines in strings
}

// NewParser creates a parser. Use strict=false for LLM output.
func NewParser(strict bool) *Parser {
	return &Parser{strict: strict}
}

// Parse attempts to repair incomplete JSON and return valid JSON.
func (p *Parser) Parse(data []byte) (*ParseResult, error) {
	if len(data) == 0 {
		return &ParseResult{
			Repaired:    []byte("{}"),
			Incomplete:  [][]string{},
			TruncatedAt: "complete",
		}, nil
	}

	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return &ParseResult{
			Repaired:    []byte("{}"),
			Incomplete:  [][]string{},
			TruncatedAt: "complete",
		}, nil
	}

	jp := &jsonParser{
		data:   data,
		strict: p.strict,
	}

	return jp.parse()
}

type jsonParser struct {
	data   []byte
	strict bool
	pos    int

	// Result tracking
	incomplete [][]string
	path       []string
}

func (p *jsonParser) parse() (*ParseResult, error) {
	repaired, truncatedAt := p.parseValue()

	return &ParseResult{
		Repaired:    repaired,
		Incomplete:  p.incomplete,
		TruncatedAt: truncatedAt,
	}, nil
}

// parseValue parses any JSON value and returns repaired bytes + truncation type
func (p *jsonParser) parseValue() ([]byte, string) {
	p.skipWhitespace()

	if p.pos >= len(p.data) {
		return []byte("null"), "value"
	}

	switch p.data[p.pos] {
	case '{':
		return p.parseObject()
	case '[':
		return p.parseArray()
	case '"':
		return p.parseString()
	case 't', 'f':
		return p.parseBoolean()
	case 'n':
		return p.parseNull()
	default:
		if p.data[p.pos] == '-' || (p.data[p.pos] >= '0' && p.data[p.pos] <= '9') {
			return p.parseNumber()
		}
		// Unknown character - return null
		return []byte("null"), "value"
	}
}

func (p *jsonParser) parseObject() ([]byte, string) {
	if p.pos >= len(p.data) || p.data[p.pos] != '{' {
		return []byte("{}"), "object"
	}

	result := []byte{'{'}
	p.pos++ // consume '{'
	truncatedAt := "complete"
	first := true

	for {
		p.skipWhitespace()

		if p.pos >= len(p.data) {
			truncatedAt = "object"
			break
		}

		if p.data[p.pos] == '}' {
			p.pos++
			result = append(result, '}')
			return result, truncatedAt
		}

		if !first {
			if p.data[p.pos] != ',' {
				truncatedAt = "object"
				break
			}
			p.pos++ // consume ','
			p.skipWhitespace()
		}
		first = false

		// Parse key
		if p.pos >= len(p.data) || p.data[p.pos] != '"' {
			truncatedAt = "key"
			break
		}

		keyBytes, keyTrunc := p.parseString()
		if keyTrunc != "complete" {
			// Incomplete key - don't include it
			p.markIncomplete("key")
			truncatedAt = keyTrunc
			break
		}
		keyStr := string(keyBytes[1 : len(keyBytes)-1]) // Remove quotes
		p.path = append(p.path, keyStr)

		p.skipWhitespace()

		// Parse colon
		if p.pos >= len(p.data) || p.data[p.pos] != ':' {
			p.markIncomplete("key")
			p.path = p.path[:len(p.path)-1]
			truncatedAt = "key"
			break
		}
		p.pos++ // consume ':'
		p.skipWhitespace()

		// Check if there's actually a value
		if p.pos >= len(p.data) {
			p.markIncomplete("value")
			p.path = p.path[:len(p.path)-1]
			truncatedAt = "value"
			break
		}

		// Parse value
		valueBytes, valueTrunc := p.parseValue()

		// Add this key-value pair to result
		if len(result) > 1 {
			result = append(result, ',')
		}
		result = append(result, keyBytes...)
		result = append(result, ':')
		result = append(result, valueBytes...)

		if valueTrunc != "complete" {
			truncatedAt = valueTrunc
			// Don't break - we've already added the partial value
		}

		p.path = p.path[:len(p.path)-1]

		if valueTrunc != "complete" {
			break
		}
	}

	result = append(result, '}')
	return result, truncatedAt
}

func (p *jsonParser) parseArray() ([]byte, string) {
	if p.pos >= len(p.data) || p.data[p.pos] != '[' {
		return []byte("[]"), "array"
	}

	result := []byte{'['}
	p.pos++ // consume '['
	truncatedAt := "complete"
	first := true
	index := 0

	for {
		p.skipWhitespace()

		if p.pos >= len(p.data) {
			truncatedAt = "array"
			break
		}

		if p.data[p.pos] == ']' {
			p.pos++
			result = append(result, ']')
			return result, truncatedAt
		}

		if !first {
			if p.data[p.pos] != ',' {
				truncatedAt = "array"
				break
			}
			p.pos++ // consume ','
			p.skipWhitespace()
		}
		first = false

		// Check if there's actually a value
		if p.pos >= len(p.data) {
			truncatedAt = "array"
			break
		}

		p.path = append(p.path, indexPath(index))
		valueBytes, valueTrunc := p.parseValue()

		// Add to result
		if len(result) > 1 {
			result = append(result, ',')
		}
		result = append(result, valueBytes...)

		if valueTrunc != "complete" {
			truncatedAt = valueTrunc
		}

		p.path = p.path[:len(p.path)-1]
		index++

		if valueTrunc != "complete" {
			break
		}
	}

	result = append(result, ']')
	return result, truncatedAt
}

func (p *jsonParser) parseString() ([]byte, string) {
	if p.pos >= len(p.data) || p.data[p.pos] != '"' {
		return []byte(`""`), "string"
	}

	start := p.pos
	p.pos++ // consume opening '"'

	for p.pos < len(p.data) {
		ch := p.data[p.pos]

		if ch == '\\' {
			p.pos++
			if p.pos >= len(p.data) {
				// Incomplete escape - escape the backslash and close the string
				p.markIncomplete("string")
				// Return string up to (but not including) the backslash, then escape it and close
				return append(p.data[start:p.pos-1], '\\', '\\', '"'), "string"
			}
			esc := p.data[p.pos]
			if esc == 'u' {
				// Unicode escape needs 4 hex digits
				p.pos++
				hexCount := 0
				for i := 0; i < 4 && p.pos < len(p.data); i++ {
					if !isHexDigit(p.data[p.pos]) {
						// Invalid or incomplete unicode escape
						if hexCount == 0 {
							// No valid hex digits - mark incomplete
							p.markIncomplete("string")
							return append(p.data[start:p.pos-2], '\\', '\\', '"'), "string"
						}
						break
					}
					p.pos++
					hexCount++
				}
				if hexCount < 4 {
					// Incomplete unicode escape - escape the backslash and close
					p.markIncomplete("string")
					return append(p.data[start:p.pos-hexCount-2], '\\', '\\', '"'), "string"
				}
			} else {
				p.pos++
			}
			continue
		}

		if ch == '"' {
			p.pos++ // consume closing '"'
			return p.data[start:p.pos], "complete"
		}

		// Handle newlines in non-strict mode
		if ch == '\n' && p.strict {
			p.markIncomplete("string")
			return append(p.data[start:p.pos], '"'), "string"
		}

		_, size := utf8.DecodeRune(p.data[p.pos:])
		p.pos += size
	}

	// String not closed
	p.markIncomplete("string")
	return append(p.data[start:p.pos], '"'), "string"
}

func (p *jsonParser) parseNumber() ([]byte, string) {
	start := p.pos

	// Optional minus
	if p.pos < len(p.data) && p.data[p.pos] == '-' {
		p.pos++
	}

	if p.pos >= len(p.data) {
		p.markIncomplete("value")
		return []byte("0"), "value"
	}

	// Integer part
	if p.data[p.pos] == '0' {
		p.pos++
	} else if p.data[p.pos] >= '1' && p.data[p.pos] <= '9' {
		for p.pos < len(p.data) && isDigit(p.data[p.pos]) {
			p.pos++
		}
	} else {
		return []byte("0"), "value"
	}

	// Decimal part
	if p.pos < len(p.data) && p.data[p.pos] == '.' {
		p.pos++
		hasDigit := false
		for p.pos < len(p.data) && isDigit(p.data[p.pos]) {
			p.pos++
			hasDigit = true
		}
		if !hasDigit {
			// Truncated after decimal point - return what we have without the dot
			p.markIncomplete("value")
			return p.data[start : p.pos-1], "value"
		}
	}

	// Exponent part
	if p.pos < len(p.data) && (p.data[p.pos] == 'e' || p.data[p.pos] == 'E') {
		expStart := p.pos
		p.pos++
		if p.pos < len(p.data) && (p.data[p.pos] == '+' || p.data[p.pos] == '-') {
			p.pos++
		}
		hasDigit := false
		for p.pos < len(p.data) && isDigit(p.data[p.pos]) {
			p.pos++
			hasDigit = true
		}
		if !hasDigit {
			// Truncated exponent - return without it
			p.markIncomplete("value")
			return p.data[start:expStart], "value"
		}
	}

	return p.data[start:p.pos], "complete"
}

func (p *jsonParser) parseBoolean() ([]byte, string) {
	if p.pos >= len(p.data) {
		return []byte("false"), "value"
	}

	var expected string
	if p.data[p.pos] == 't' {
		expected = "true"
	} else {
		expected = "false"
	}

	for i := 0; i < len(expected); i++ {
		if p.pos >= len(p.data) || p.data[p.pos] != expected[i] {
			p.markIncomplete("value")
			return []byte(expected), "value"
		}
		p.pos++
	}

	return []byte(expected), "complete"
}

func (p *jsonParser) parseNull() ([]byte, string) {
	expected := "null"
	for i := 0; i < len(expected); i++ {
		if p.pos >= len(p.data) || p.data[p.pos] != expected[i] {
			p.markIncomplete("value")
			return []byte("null"), "value"
		}
		p.pos++
	}
	return []byte("null"), "complete"
}

func (p *jsonParser) markIncomplete(reason string) {
	if len(p.path) > 0 {
		pathCopy := make([]string, len(p.path))
		copy(pathCopy, p.path)
		p.incomplete = append(p.incomplete, pathCopy)
	}
}

func (p *jsonParser) skipWhitespace() {
	for p.pos < len(p.data) {
		ch := p.data[p.pos]
		if ch != ' ' && ch != '\t' && ch != '\n' && ch != '\r' {
			break
		}
		p.pos++
	}
}

func indexPath(i int) string {
	return "[" + itoa(i) + "]"
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(buf[pos:])
}

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

func isHexDigit(ch byte) bool {
	return isDigit(ch) || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')
}
