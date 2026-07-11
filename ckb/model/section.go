package model

// Section represents a parsed section in the Markdown body, keyed by its heading.
type Section struct {
	Heading string `json:"heading"`
	Body    string `json:"body"`
}
