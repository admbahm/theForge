// Package importer converts a conventional Markdown master resume into a
// reviewable Career Knowledge Base without modifying the source document.
package importer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/admbahm/theForge/ckb/model"
	"github.com/admbahm/theForge/ckb/parser"
)

var (
	headingPattern           = regexp.MustCompile(`^(#{1,6})\s+(.+?)\s*$`)
	emailPattern             = regexp.MustCompile(`(?i)[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}`)
	phonePattern             = regexp.MustCompile(`(?:\+?1[-.\s]?)?\(?[0-9]{3}\)?[-.\s][0-9]{3}[-.\s][0-9]{4}`)
	slugPattern              = regexp.MustCompile(`[^a-z0-9]+`)
	durationSeparatorPattern = regexp.MustCompile(`\s+[-–—]\s+`)
)

type Options struct {
	SourcePath  string
	OutputDir   string
	Ready       bool
	LastUpdated time.Time
}

type Report struct {
	SchemaVersion string   `json:"schema_version"`
	SourceSHA256  string   `json:"source_sha256"`
	Ready         bool     `json:"ready_for_public_artifacts"`
	Generated     []string `json:"generated_files"`
	Experiences   int      `json:"experience_count"`
	Projects      int      `json:"project_count"`
	Warnings      []string `json:"warnings"`
}

type heading struct {
	Level int
	Title string
	Line  int
}

type section struct {
	Heading heading
	Lines   []string
}

type experience struct {
	Organization string
	Role         string
	Duration     string
	Location     string
	Mission      []string
	Achievements []string
}

// Import creates a new CKB directory atomically. Existing output is never
// overwritten; the caller must choose a new directory for every review pass.
func Import(options Options) (Report, error) {
	if strings.TrimSpace(options.SourcePath) == "" || strings.TrimSpace(options.OutputDir) == "" {
		return Report{}, fmt.Errorf("source and output paths are required")
	}
	sourcePath, err := filepath.Abs(options.SourcePath)
	if err != nil {
		return Report{}, fmt.Errorf("resolve source: %w", err)
	}
	outputDir, err := filepath.Abs(options.OutputDir)
	if err != nil {
		return Report{}, fmt.Errorf("resolve output: %w", err)
	}
	if _, err := os.Stat(outputDir); err == nil {
		return Report{}, fmt.Errorf("output directory already exists: %s", outputDir)
	} else if !os.IsNotExist(err) {
		return Report{}, fmt.Errorf("inspect output directory: %w", err)
	}
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		return Report{}, fmt.Errorf("read master resume: %w", err)
	}
	if len(source) == 0 {
		return Report{}, fmt.Errorf("master resume is empty")
	}

	text, contactWarnings := removeContactPII(string(source))
	sections := parseSections(text)
	experiences := parseExperiences(sections)
	if len(experiences) == 0 {
		return Report{}, fmt.Errorf("no roles found under a PROFESSIONAL EXPERIENCE heading")
	}

	date := options.LastUpdated
	if date.IsZero() {
		date = time.Now()
	}
	status := "Draft"
	if options.Ready {
		status = "Active"
	}
	metadata := metadataOptions{Status: status, Date: date.Format("2006-01-02")}
	files := make(map[string][]byte)
	for _, item := range experiences {
		name := slug(item.Organization + " " + item.Role)
		path := filepath.ToSlash(filepath.Join("experience", name+".md"))
		files[path] = []byte(renderExperience(item, "exp:"+name, metadata))
	}

	if content := renderSkills(sections, metadata); content != "" {
		files["skills.md"] = []byte(content)
	}
	projects := parseNamedSections(sections, "PLATFORM, CLOUD & AI PROJECTS")
	for _, project := range projects {
		name := slug(project.Heading.Title)
		files[filepath.ToSlash(filepath.Join("projects", name+".md"))] = []byte(renderProject(project, "proj:"+name, metadata))
	}
	if content := renderEducation(sections, metadata); content != "" {
		files["education.md"] = []byte(content)
	}
	if content := renderCredentials(sections, metadata); content != "" {
		files["credentials.md"] = []byte(content)
	}
	if content := renderProfile(sections, metadata); content != "" {
		files["profile.md"] = []byte(content)
	}

	digest := sha256.Sum256(source)
	report := Report{
		SchemaVersion: "1.0",
		SourceSHA256:  hex.EncodeToString(digest[:]),
		Ready:         options.Ready,
		Experiences:   len(experiences),
		Projects:      len(projects),
		Warnings:      contactWarnings,
	}
	for path := range files {
		report.Generated = append(report.Generated, path)
	}
	sort.Strings(report.Generated)
	if !options.Ready {
		report.Warnings = append(report.Warnings, "Imported records are Draft and cannot be used by StrictPublic artifact generation; review them and rerun with --ready into a new directory.")
	}

	parent := filepath.Dir(outputDir)
	if err := os.MkdirAll(parent, 0o700); err != nil {
		return Report{}, fmt.Errorf("create output parent: %w", err)
	}
	stage, err := os.MkdirTemp(parent, ".ckb-import-")
	if err != nil {
		return Report{}, fmt.Errorf("create import staging directory: %w", err)
	}
	defer os.RemoveAll(stage)
	if err := os.Chmod(stage, 0o700); err != nil {
		return Report{}, err
	}
	for path, content := range files {
		fullPath := filepath.Join(stage, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o700); err != nil {
			return Report{}, err
		}
		if err := writeSyncedFile(fullPath, content); err != nil {
			return Report{}, err
		}
	}
	reportBytes, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return Report{}, err
	}
	if err := writeSyncedFile(filepath.Join(stage, "import-report.json"), append(reportBytes, '\n')); err != nil {
		return Report{}, err
	}
	validationResult := parser.ParseDirectory(context.Background(), stage, parser.ParseOptions{
		Strict:          true,
		ValidatePrivacy: true,
		Limits:          parser.DefaultLimits(),
	})
	if model.HasBlockingDiagnostics(validationResult.Diagnostics) {
		codes := model.BlockingDiagnosticCodes(validationResult.Diagnostics)
		return Report{}, fmt.Errorf("generated CKB failed validation with diagnostic code(s): %v", codes)
	}
	for _, directory := range []string{filepath.Join(stage, "experience"), filepath.Join(stage, "projects"), stage} {
		if _, err := os.Stat(directory); os.IsNotExist(err) {
			continue
		}
		if err := syncDirectory(directory); err != nil {
			return Report{}, fmt.Errorf("sync imported CKB directory: %w", err)
		}
	}
	if err := os.Rename(stage, outputDir); err != nil {
		return Report{}, fmt.Errorf("publish imported CKB: %w", err)
	}
	if err := syncDirectory(parent); err != nil {
		return Report{}, fmt.Errorf("sync imported CKB parent: %w", err)
	}
	return report, nil
}

func writeSyncedFile(path string, data []byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err := file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}

func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}

func removeContactPII(text string) (string, []string) {
	warnings := []string{}
	if emailPattern.MatchString(text) {
		text = emailPattern.ReplaceAllString(text, "[CONTACT REMOVED]")
		warnings = append(warnings, "Email address removed; configure THEFORGE_CONTACT_EMAIL separately.")
	}
	if phonePattern.MatchString(text) {
		text = phonePattern.ReplaceAllString(text, "[CONTACT REMOVED]")
		warnings = append(warnings, "Phone number removed; configure THEFORGE_CONTACT_PHONE separately.")
	}
	return text, warnings
}

func parseSections(text string) []section {
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	var result []section
	current := section{Heading: heading{Level: 0, Title: "Document", Line: 1}}
	for index, line := range lines {
		match := headingPattern.FindStringSubmatch(strings.TrimSpace(line))
		if match == nil {
			current.Lines = append(current.Lines, strings.TrimRight(line, " \t"))
			continue
		}
		if current.Heading.Level != 0 || hasContent(current.Lines) {
			result = append(result, current)
		}
		current = section{Heading: heading{Level: len(match[1]), Title: cleanHeading(match[2]), Line: index + 1}}
	}
	result = append(result, current)
	return result
}

func parseExperiences(sections []section) []experience {
	start, end := sectionRange(sections, "PROFESSIONAL EXPERIENCE")
	var result []experience
	var organization string
	roleCaptured := false
	for index := start; index < end; index++ {
		sec := sections[index]
		if sec.Heading.Level == 2 {
			organization = sec.Heading.Title
			roleCaptured = false
			continue
		}
		if sec.Heading.Level != 3 || organization == "" || roleCaptured {
			continue
		}
		roleCaptured = true
		item := experience{Organization: organization, Role: sec.Heading.Title}
		plain := contentLines(sec.Lines)
		if len(plain) > 0 {
			item.Duration = normalizeDuration(plain[0])
		}
		if len(plain) > 1 && looksLikeLocation(plain[1]) {
			item.Location = plain[1]
			plain = plain[2:]
		} else if len(plain) > 0 {
			plain = plain[1:]
		}
		item.Mission = append(item.Mission, plain...)
		for next := index + 1; next < end; next++ {
			if sections[next].Heading.Level <= 2 {
				break
			}
			item.Achievements = append(item.Achievements, bulletLines(sections[next].Lines)...)
		}
		result = append(result, item)
	}
	return result
}

func parseNamedSections(sections []section, title string) []section {
	start, end := sectionRange(sections, title)
	var result []section
	for index := start; index < end; index++ {
		if sections[index].Heading.Level != 2 {
			continue
		}
		combined := sections[index]
		for next := index + 1; next < end && sections[next].Heading.Level > 2; next++ {
			combined.Lines = append(combined.Lines, sections[next].Lines...)
		}
		result = append(result, combined)
	}
	return result
}

type metadataOptions struct {
	Status string
	Date   string
}

func metadata(id, objectType, lifecycle string, options metadataOptions) string {
	return fmt.Sprintf(`| Metadata | Value |
| :--- | :--- |
| **Schema Version** | 1.0 |
| **ID** | %s |
| **Type** | %s |
| **Status** | %s |
| **Verification Level** | Self-Attested |
| **Confidence** | 0.85 |
| **Visibility** | Public |
| **Source** | Imported Master Resume |
| **Last Updated** | %s |
| **Lifecycle State** | %s |

---

`, id, objectType, options.Status, options.Date, lifecycle)
}

func renderExperience(item experience, id string, options metadataOptions) string {
	var output strings.Builder
	output.WriteString(metadata(id, "Experience", "Completed", options))
	output.WriteString("## 1. Role Context\n")
	fmt.Fprintf(&output, "* **Organization**: %s\n* **Role**: %s\n", item.Organization, item.Role)
	if item.Duration != "" {
		fmt.Fprintf(&output, "* **Duration**: %s\n", item.Duration)
	}
	if item.Location != "" {
		fmt.Fprintf(&output, "* **Location**: %s\n", item.Location)
	}
	if len(item.Mission) > 0 {
		fmt.Fprintf(&output, "* **Mission**: %s\n", strings.Join(item.Mission, " "))
	}
	output.WriteString("\n---\n\n## 2. Key Achievements\n\n### Imported Achievements\n")
	for _, achievement := range item.Achievements {
		fmt.Fprintf(&output, "* %s\n", achievement)
	}
	return output.String()
}

func renderSkills(sections []section, options metadataOptions) string {
	start, end := sectionRange(sections, "CORE COMPETENCIES")
	if start == end {
		return ""
	}
	var output strings.Builder
	output.WriteString(metadata("skill:main", "Skill", "Active", options))
	output.WriteString("## 1. Skill Matrix by Domain\n")
	for index := start; index < end; index++ {
		sec := sections[index]
		if sec.Heading.Level != 2 {
			continue
		}
		values := splitSkills(sec.Lines)
		if len(values) == 0 {
			continue
		}
		fmt.Fprintf(&output, "\n### %s\n\n", sec.Heading.Title)
		for _, value := range values {
			fmt.Fprintf(&output, "* %s\n", strings.ReplaceAll(value, "|", "/"))
		}
	}
	return output.String()
}

func renderProject(sec section, id string, options metadataOptions) string {
	content := strings.Join(contentLines(sec.Lines), "\n")
	return metadata(id, "Project", "Active", options) +
		"## 1. Project Specifications\n* **Objective**: " + sec.Heading.Title + "\n* **Source Detail**: " + content +
		"\n\n---\n\n## 2. Architecture & Design Decisions\n* Imported source statements require review.\n" +
		"\n---\n\n## 3. Implementation Details\n* " + content +
		"\n\n---\n\n## 4. Outcomes & Metrics\n* Only metrics explicitly present in the imported source detail are authorized.\n"
}

func renderEducation(sections []section, options metadataOptions) string {
	lines := topSectionContent(sections, "EDUCATION")
	if len(lines) == 0 {
		return ""
	}
	var output strings.Builder
	output.WriteString(metadata("edu:main", "Education", "Completed", options))
	output.WriteString("## 1. Academic Credentials\n\n### Imported Education\n")
	for _, line := range lines {
		fmt.Fprintf(&output, "* %s\n", line)
	}
	return output.String()
}

func renderCredentials(sections []section, options metadataOptions) string {
	lines := topSectionContent(sections, "CERTIFICATIONS")
	if len(lines) == 0 {
		return ""
	}
	var output strings.Builder
	output.WriteString(metadata("cred:main", "Credential", "Active", options))
	output.WriteString("## 1. Professional Certifications\n")
	for index, line := range lines {
		fmt.Fprintf(&output, "\n### Imported Credential %d\n* **Name**: %s\n* **Issue Status**: Active\n", index+1, line)
	}
	output.WriteString("\n---\n\n## 2. Professional Training Log\n\nNo training records were imported.\n")
	return output.String()
}

func renderProfile(sections []section, options metadataOptions) string {
	summary := topSectionContent(sections, "PROFESSIONAL SUMMARY")
	if len(summary) == 0 {
		return ""
	}
	text := strings.Join(summary, " ")
	return metadata("profile:main", "Profile", "Active", options) +
		"## 1. Professional Vision\n" + text +
		"\n\n---\n\n## 2. Core Target Profile\n\nImported from the master resume; review target roles before activation.\n" +
		"\n---\n\n## 3. Technology Alignment Priorities\n\nSee `skills.md`.\n" +
		"\n---\n\n## 4. Career Constraints & Non-Negotiables\n\nNo constraints were inferred from the master resume.\n"
}

func sectionRange(sections []section, title string) (int, int) {
	start := -1
	level := 0
	for index, sec := range sections {
		if start < 0 && strings.EqualFold(sec.Heading.Title, title) {
			start, level = index+1, sec.Heading.Level
			continue
		}
		if start >= 0 && sec.Heading.Level <= level {
			return start, index
		}
	}
	if start < 0 {
		return 0, 0
	}
	return start, len(sections)
}

func topSectionContent(sections []section, title string) []string {
	start, end := sectionRange(sections, title)
	var lines []string
	for index := start; index < end; index++ {
		lines = append(lines, contentLines(sections[index].Lines)...)
	}
	return lines
}

func contentLines(lines []string) []string {
	var result []string
	for _, line := range lines {
		line = strings.TrimSpace(strings.TrimSuffix(line, "  "))
		line = strings.TrimSpace(strings.TrimLeft(line, "*-"))
		if line == "" || line == "---" {
			continue
		}
		result = append(result, line)
	}
	return result
}

func bulletLines(lines []string) []string {
	var result []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "*") || strings.HasPrefix(trimmed, "-") {
			result = append(result, strings.TrimSpace(strings.TrimLeft(trimmed, "*-")))
		}
	}
	return result
}

func splitSkills(lines []string) []string {
	seen := make(map[string]struct{})
	var values []string
	for _, line := range contentLines(lines) {
		for _, value := range strings.Split(line, ",") {
			value = strings.TrimSpace(strings.Trim(value, "•;"))
			if value == "" {
				continue
			}
			key := strings.ToLower(value)
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			values = append(values, value)
		}
	}
	return values
}

func cleanHeading(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, "*_` ")
	return strings.TrimSpace(value)
}

func slug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = slugPattern.ReplaceAllString(value, "-")
	return strings.Trim(value, "-")
}

func hasContent(lines []string) bool { return len(contentLines(lines)) > 0 }

func looksLikeLocation(value string) bool {
	lower := strings.ToLower(value)
	return strings.Contains(lower, "remote") || strings.Contains(value, ",") || strings.Contains(value, "/")
}

func normalizeDuration(value string) string {
	parts := durationSeparatorPattern.Split(strings.TrimSpace(value), 2)
	if len(parts) != 2 {
		return value
	}
	start, ok := normalizeDatePart(parts[0])
	if !ok {
		return value
	}
	endText := strings.TrimSpace(parts[1])
	if strings.EqualFold(endText, "present") || strings.EqualFold(endText, "current") {
		return start + " – Present"
	}
	end, ok := normalizeDatePart(endText)
	if !ok {
		return value
	}
	return start + " – " + end
}

func normalizeDatePart(value string) (string, bool) {
	value = strings.TrimSpace(value)
	for _, layout := range []string{"January 2006", "Jan 2006", "2006-01", "2006"} {
		parsed, err := time.Parse(layout, value)
		if err != nil {
			continue
		}
		if layout == "2006" {
			return parsed.Format("2006"), true
		}
		return parsed.Format("2006-01"), true
	}
	return "", false
}
