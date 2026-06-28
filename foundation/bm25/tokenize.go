package bm25

import "strings"

// tokenize splits text into lowercase terms.
// CJK characters are individual tokens; Latin text is split on delimiters.
func tokenize(text string) []string {
	text = strings.ToLower(text)
	var terms []string
	var current strings.Builder

	for _, r := range text {
		if isCJK(r) {
			if current.Len() > 0 {
				terms = append(terms, current.String())
				current.Reset()
			}
			terms = append(terms, string(r))
		} else if isTokenDelimiter(r) {
			if current.Len() > 0 {
				terms = append(terms, current.String())
				current.Reset()
			}
		} else {
			current.WriteRune(r)
		}
	}

	if current.Len() > 0 {
		terms = append(terms, current.String())
	}

	return terms
}

// isCJK checks if a rune is a CJK character (Chinese, Japanese, Korean).
func isCJK(r rune) bool {
	return (r >= 0x4E00 && r <= 0x9FFF) ||
		(r >= 0x3400 && r <= 0x4DBF) ||
		(r >= 0x20000 && r <= 0x2A6DF) ||
		(r >= 0x2A700 && r <= 0x2B73F) ||
		(r >= 0x2B740 && r <= 0x2B81F) ||
		(r >= 0xF900 && r <= 0xFAFF) ||
		(r >= 0x2F800 && r <= 0x2FA1F)
}

// isTokenDelimiter checks if a rune is a token delimiter.
func isTokenDelimiter(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\r' ||
		r == '.' || r == ',' || r == ';' || r == ':' ||
		r == '!' || r == '?' || r == '"' || r == '\'' ||
		r == '(' || r == ')' || r == '[' || r == ']' ||
		r == '{' || r == '}' || r == '-' || r == '_'
}
