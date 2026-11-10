package journal

import (
	"encoding/json"
	"strings"
	"unicode/utf8"
)

const (
	maxBodyDepth = 50  // real documents (lists in quotes in lists) stay far below this
	excerptRunes = 200 // list-view preview length
)

// tiptapNode is the part of a Tiptap node the API reads. Marks and attrs stay in the stored
// body but are ignored here.
type tiptapNode struct {
	Type    string       `json:"type"`
	Text    string       `json:"text"`
	Content []tiptapNode `json:"content"`
}

// ParseBody checks that body is a Tiptap document and returns its plain text.
func ParseBody(body json.RawMessage) (string, error) {
	if len(body) == 0 {
		return "", &ValidationError{Field: "body", Message: "is required"}
	}
	if len(body) > MaxBodyBytes {
		return "", &ValidationError{Field: "body", Message: "must be 256 KB or smaller"}
	}
	var doc tiptapNode
	if err := json.Unmarshal(body, &doc); err != nil || doc.Type != "doc" {
		return "", &ValidationError{Field: "body", Message: `must be a Tiptap document like {"type":"doc","content":[...]}`}
	}
	var b strings.Builder
	if !collectText(&b, doc, 0) {
		return "", &ValidationError{Field: "body", Message: "is nested too deeply"}
	}
	return b.String(), nil
}

// collectText appends n's text to b, depth first. It returns false if the tree is too deep.
func collectText(b *strings.Builder, n tiptapNode, depth int) bool {
	if depth > maxBodyDepth {
		return false
	}
	b.WriteString(n.Text)
	for _, child := range n.Content {
		if !collectText(b, child, depth+1) {
			return false
		}
	}
	if n.Type != "text" {
		b.WriteByte(' ') // paragraphs, headings and line breaks separate words
	}
	return true
}

// Summarise returns the list-view excerpt (first 200 characters) and the word count of text.
func Summarise(text string) (excerpt string, words int) {
	fields := strings.Fields(text)
	excerpt = strings.Join(fields, " ")
	if utf8.RuneCountInString(excerpt) > excerptRunes {
		excerpt = strings.TrimSpace(string([]rune(excerpt)[:excerptRunes])) + "…"
	}
	return excerpt, len(fields)
}
