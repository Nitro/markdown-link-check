package parser

import (
	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday/v2"
	"github.com/shurcooL/sanitized_anchor_name"
)

// Markdown expose a parser that transform Markdown into HTML.
type Markdown struct {
	policy bluemonday.Policy
}

// Init the internal state.
func (m *Markdown) Init() {
	m.policy = *bluemonday.UGCPolicy()
}

// Do transform the Markdown into HTML.
func (m Markdown) Do(payload []byte) []byte {
	payload = blackfriday.Run(payload, blackfriday.WithExtensions(blackfriday.AutoHeadingIDs))
	return m.policy.SanitizeBytes(payload)
}

// SanitizedAnchorName process the anchor.
func (m Markdown) SanitizedAnchorName(text string) string {
	return sanitized_anchor_name.Create(text)
}
