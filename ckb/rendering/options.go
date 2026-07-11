package rendering

import (
	"github.com/admbahm/theForge/ckb/planning"
)

// ContactInfo represents layout-level presentation details supplied at render time.
type ContactInfo struct {
	Name        string `json:"name,omitempty"`
	Phone       string `json:"phone,omitempty"`
	Email       string `json:"email,omitempty"`
	Address     string `json:"address,omitempty"`
	LinkedInURL string `json:"linkedin_url,omitempty"`
	GitHubURL   string `json:"github_url,omitempty"`
}

// RenderOptions configures deterministic adaptation and formatting behaviors of the renderer.
type RenderOptions struct {
	Subtype                string      `json:"subtype,omitempty"`                  // "professional", "technical", "executive", "academic-adjacent"
	BiographyLength        string      `json:"biography_length,omitempty"`         // "short", "medium", "long"
	PronounStyle           string      `json:"pronoun_style,omitempty"`            // "neutral", "he/him", "she/her", "they/them"
	IncludeHeadings        bool        `json:"include_headings,omitempty"`         // default true
	IncludeWarnings        bool        `json:"include_warnings,omitempty"`         // default true
	DebugProvenance        bool        `json:"debug_provenance,omitempty"`         // default false
	DateStyle              string      `json:"date_style,omitempty"`               // "Standard", "YearOnly"
	MetricStyle            string      `json:"metric_style,omitempty"`             // "Standard", "Compact"
	AbbreviationStyle      string      `json:"abbreviation_style,omitempty"`       // "Standard", "Expanded"
	MaxSummaryStatements   int         `json:"max_summary_statements,omitempty"`   // custom limits
	IncompleteSTARBehavior string      `json:"incomplete_star_behavior,omitempty"` // "Fail", "WarnAndRender", "Skip"
	Contact                ContactInfo `json:"contact,omitempty"`
}

// RenderRequest coordinates the inputs for the renderer.
type RenderRequest struct {
	Plan    *planning.ArtifactPlan `json:"plan"`
	Options RenderOptions          `json:"options"`
}
