package claims

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/admbahm/theForge/ckb/model"
)

var idReferenceRegex = regexp.MustCompile(`[a-z0-9]+:[a-z0-9-]+`)

// ExtractClaims walks the KnowledgeBase and extracts normalized claims.
func ExtractClaims(kb *model.KnowledgeBase) ([]Claim, error) {
	var claims []Claim

	// Process in stable alphabetical order of object IDs
	var keys []string
	for k := range kb.Objects {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		obj := kb.Objects[k]
		extracted, err := extractFromObject(obj)
		if err != nil {
			return nil, err
		}
		claims = append(claims, extracted...)
	}

	return claims, nil
}

func extractFromObject(obj *model.Object) ([]Claim, error) {
	var claims []Claim

	switch obj.Type {
	case model.TypeExperience:
		claims = append(claims, extractFromExperience(obj)...)
	case model.TypeProject:
		claims = append(claims, extractFromProject(obj)...)
	case model.TypeEducation:
		claims = append(claims, extractFromEducation(obj)...)
	case model.TypeCredential:
		claims = append(claims, extractFromCredential(obj)...)
	case model.TypeContribution:
		claims = append(claims, extractFromContribution(obj)...)
	case model.TypeSkill:
		claims = append(claims, extractFromSkill(obj)...)
	case model.TypeAccomplishment:
		claims = append(claims, extractFromAccomplishment(obj)...)
	}

	return claims, nil
}

func extractFromExperience(obj *model.Object) []Claim {
	var claims []Claim

	// Parse duration, role, and details from Role Context
	for _, sec := range obj.Sections {
		heading := strings.ToLower(sec.Heading)
		if strings.Contains(heading, "role context") || strings.Contains(heading, "context") {
			kvs := parseKeyValues(sec.Body)
			role, hasRole := kvs["Role"]
			duration, hasDuration := kvs["Duration"]
			location := kvs["Location"]
			organization := extractExperienceOrganization(kvs)

			// 1. Role Claim
			if hasRole {
				stmt := fmt.Sprintf("Served as %s", role)
				if organization != "" {
					stmt = fmt.Sprintf("%s at %s", stmt, organization)
				}
				if location != "" {
					stmt = fmt.Sprintf("%s (%s)", stmt, location)
				}
				c := buildBaseClaim(obj, KindRole, stmt, ClaimSubject(obj.ID), "served_as", ClaimValue(role), parseTimeRange(duration))
				addOrganization(&c, organization)
				claims = append(claims, c)
			}

			// 2. Employment/Chronology Claim
			if hasDuration {
				stmt := fmt.Sprintf("Employment during %s", duration)
				if organization != "" {
					stmt = fmt.Sprintf("Employed at %s during %s", organization, duration)
				}
				c := buildBaseClaim(obj, KindEmployment, stmt, ClaimSubject(obj.ID), "employed_at", ClaimValue(organization), parseTimeRange(duration))
				addOrganization(&c, organization)
				claims = append(claims, c)
			}

			// 3. Responsibilities
			lines := strings.Split(sec.Body, "\n")
			inResp := false
			for _, line := range lines {
				trimmed := strings.TrimSpace(line)
				if strings.Contains(strings.ToLower(trimmed), "responsibilities") {
					inResp = true
					continue
				}
				if inResp {
					if trimmed == "" {
						continue
					}
					if strings.Contains(trimmed, ":") && (strings.HasPrefix(trimmed, "**") || strings.HasPrefix(trimmed, "*")) {
						inResp = false
						continue
					}
					stmt := fmt.Sprintf("Responsible for: %s", trimmed)
					c := buildBaseClaim(obj, KindResponsibility, stmt, ClaimSubject(obj.ID), "responsible_for", ClaimValue(trimmed), parseTimeRange(duration))
					addOrganization(&c, organization)
					claims = append(claims, c)
				}
			}
		}

		// 4. Key Achievements (Accomplishments / Measurable Results / Gaps)
		if strings.Contains(heading, "achievements") || strings.Contains(heading, "outcomes") ||
			strings.Contains(heading, "technical") || strings.Contains(heading, "architectural") ||
			strings.Contains(heading, "leadership") || strings.Contains(heading, "impact") ||
			strings.Contains(heading, "metrics") || strings.Contains(heading, "accomplishments") {
			bullets := parseBulletList(sec.Body)
			for _, bullet := range bullets {
				kind := KindAccomplishment
				metrics := parseMetrics(bullet)
				hasApproved := false
				for _, m := range metrics {
					if m.Status == "Approved" && !m.IsAmbiguous {
						hasApproved = true
					}
				}
				if hasApproved {
					kind = KindMeasurableResult
				}
				c := buildBaseClaim(obj, kind, bullet, ClaimSubject(obj.ID), "accomplished", ClaimValue(bullet), nil)
				c.Metrics = metrics
				claims = append(claims, c)
			}
		}

		// 5. Skills developed
		if strings.Contains(heading, "technologies") || strings.Contains(heading, "skills") {
			kvs := parseKeyValues(sec.Body)
			if tech, ok := kvs["Technologies"]; ok {
				skills := strings.Split(tech, ",")
				for _, sk := range skills {
					skTrim := strings.TrimSpace(sk)
					if skTrim != "" {
						stmt := fmt.Sprintf("Utilized technology: %s", skTrim)
						c := buildBaseClaim(obj, KindTechnicalSkill, stmt, ClaimSubject(obj.ID), "skilled_in", ClaimValue(skTrim), nil)
						c.Skills = []string{skTrim}
						claims = append(claims, c)
					}
				}
			}
			if meth, ok := kvs["Methodologies"]; ok {
				skills := strings.Split(meth, ",")
				for _, sk := range skills {
					skTrim := strings.TrimSpace(sk)
					if skTrim != "" {
						stmt := fmt.Sprintf("Practiced methodology: %s", skTrim)
						c := buildBaseClaim(obj, KindLeadershipSkill, stmt, ClaimSubject(obj.ID), "proficient_in", ClaimValue(skTrim), nil)
						c.Skills = []string{skTrim}
						claims = append(claims, c)
					}
				}
			}
		}
	}

	return claims
}

func extractFromProject(obj *model.Object) []Claim {
	var claims []Claim

	for _, sec := range obj.Sections {
		heading := strings.ToLower(sec.Heading)
		if strings.Contains(heading, "specifications") || strings.Contains(heading, "objective") {
			kvs := parseKeyValues(sec.Body)
			if objv, ok := kvs["Objective"]; ok {
				stmt := fmt.Sprintf("Project Objective: %s", objv)
				claims = append(claims, buildBaseClaim(obj, KindProjectContribution, stmt, ClaimSubject(obj.ID), "objective_of", ClaimValue(objv), nil))
			}
			if sol, ok := kvs["Solution"]; ok {
				stmt := fmt.Sprintf("Project Solution: %s", sol)
				claims = append(claims, buildBaseClaim(obj, KindProjectContribution, stmt, ClaimSubject(obj.ID), "contributed_solution", ClaimValue(sol), nil))
			}
		}

		if strings.Contains(heading, "outcomes") || strings.Contains(heading, "metrics") {
			bullets := parseBulletList(sec.Body)
			for _, bullet := range bullets {
				kind := KindProjectContribution
				metrics := parseMetrics(bullet)
				if len(metrics) > 0 {
					kind = KindMeasurableResult
				}
				c := buildBaseClaim(obj, kind, bullet, ClaimSubject(obj.ID), "achieved_project_outcome", ClaimValue(bullet), nil)
				c.Metrics = metrics
				claims = append(claims, c)
			}
		}
	}

	return claims
}

func extractFromEducation(obj *model.Object) []Claim {
	var claims []Claim
	inAcademicCredentials := false

	for _, sec := range obj.Sections {
		level, title := splitHeading(sec.Heading)
		titleLower := strings.ToLower(title)

		if level == 2 {
			inAcademicCredentials = strings.Contains(titleLower, "academic credentials")
			continue
		}

		if !inAcademicCredentials || level != 3 || title == "" {
			continue
		}

		kvs := parseKeyValues(sec.Body)
		var parts []string
		parts = append(parts, "Degree: "+title)
		if institution := kvs["Institution"]; institution != "" {
			parts = append(parts, "Institution: "+institution)
		}
		if program := kvs["Program"]; program != "" {
			parts = append(parts, "Program: "+program)
		}
		if timeline := kvs["Timeline"]; timeline != "" {
			parts = append(parts, "Timeline: "+timeline)
		}
		if status := kvs["Status"]; status != "" {
			parts = append(parts, "Status: "+status)
		}

		stmt := strings.Join(parts, "; ")
		c := buildBaseClaim(obj, KindEducation, stmt, ClaimSubject(obj.ID), "education_record", ClaimValue(title), parseTimeRange(kvs["Timeline"]))
		if institution := kvs["Institution"]; institution != "" {
			c.Organizations = []string{institution}
		}
		claims = append(claims, c)
	}

	return claims
}

func splitHeading(heading string) (int, string) {
	trimmed := strings.TrimSpace(heading)
	level := 0
	for level < len(trimmed) && trimmed[level] == '#' {
		level++
	}
	if level == 0 {
		return 0, trimmed
	}
	return level, strings.TrimSpace(trimmed[level:])
}

func extractFromCredential(obj *model.Object) []Claim {
	var claims []Claim
	for _, sec := range obj.Sections {
		level, title := splitHeading(sec.Heading)
		if level != 3 || title == "" {
			continue
		}

		kvs := parseKeyValues(sec.Body)
		var parts []string
		parts = append(parts, "Credential: "+title)
		if issuer := firstNonEmpty(kvs["Issuer"], kvs["Provider"]); issuer != "" {
			parts = append(parts, "Issuer: "+issuer)
		}
		if status := firstNonEmpty(kvs["Issue Status"], kvs["Status"]); status != "" {
			parts = append(parts, "Status: "+status)
		}
		if issueDate := firstNonEmpty(kvs["Issue Date"], kvs["Completion Date"]); issueDate != "" {
			parts = append(parts, "Issue Date: "+issueDate)
		}
		if expiration := kvs["Expiration Date"]; expiration != "" {
			parts = append(parts, "Expiration Date: "+expiration)
		}
		if credentialType := kvs["Type"]; credentialType != "" {
			parts = append(parts, "Type: "+credentialType)
		}
		if competencies := firstNonEmpty(kvs["Competency / Topic"], kvs["Topics Covered"]); competencies != "" {
			parts = append(parts, "Competencies: "+competencies)
		}

		stmt := strings.Join(parts, "; ")
		c := buildBaseClaim(obj, KindCredential, stmt, ClaimSubject(obj.ID), "credential_record", ClaimValue(title), nil)
		if issuer := firstNonEmpty(kvs["Issuer"], kvs["Provider"]); issuer != "" {
			c.Organizations = []string{issuer}
		}
		if projects := extractIDs(firstNonEmpty(kvs["Related Projects"], kvs["Projects"])); len(projects) > 0 {
			c.Projects = projects
		}
		if evidence := extractIDs(firstNonEmpty(kvs["Evidence Ref"], kvs["Supporting Evidence"])); len(evidence) > 0 {
			c.EvidenceObjectIDs = evidence
		}
		if status := firstNonEmpty(kvs["Issue Status"], kvs["Status"]); status != "" {
			c.Status = status
		}
		claims = append(claims, c)
	}
	return claims
}

func extractFromContribution(obj *model.Object) []Claim {
	var claims []Claim
	for _, sec := range obj.Sections {
		heading := strings.ToLower(sec.Heading)
		bullets := parseBulletList(sec.Body)
		kind := KindPublication
		if strings.Contains(heading, "speaking") || strings.Contains(heading, "presentation") {
			kind = KindSpeaking
		}
		for _, bullet := range bullets {
			claims = append(claims, buildBaseClaim(obj, kind, bullet, ClaimSubject(obj.ID), "contributed", ClaimValue(bullet), nil))
		}
	}
	return claims
}

func extractFromSkill(obj *model.Object) []Claim {
	var claims []Claim
	seenSkills := make(map[string]bool)
	for _, sec := range obj.Sections {
		_, category := splitHeading(sec.Heading)
		if category == "" {
			category = strings.TrimSpace(sec.Heading)
		}
		for _, row := range parseSkillTableRows(sec.Body) {
			skillName := row["Skill Name"]
			if skillName == "" {
				continue
			}
			normalizedSkill := strings.ToLower(strings.Join(strings.Fields(skillName), " "))
			if seenSkills[normalizedSkill] {
				continue
			}
			seenSkills[normalizedSkill] = true

			var parts []string
			parts = append(parts, fmt.Sprintf("Utilized technology: %s", skillName))
			if category != "" && !strings.Contains(strings.ToLower(category), "skill matrix") {
				parts = append(parts, "Category: "+category)
			}
			if proficiency := row["Proficiency"]; proficiency != "" {
				parts = append(parts, "Proficiency: "+proficiency)
			}
			if years := row["Years"]; years != "" {
				parts = append(parts, "Years: "+years)
			}
			if lastUsed := row["Last Used"]; lastUsed != "" {
				parts = append(parts, "Last Used: "+lastUsed)
			}

			c := buildBaseClaim(obj, KindTechnicalSkill, strings.Join(parts, "; "), ClaimSubject(obj.ID), "skilled_in", ClaimValue(skillName), nil)
			c.Skills = []string{skillName}
			if confidence := row["Confidence"]; confidence != "" {
				if parsed, err := strconv.ParseFloat(confidence, 64); err == nil {
					c.Confidence = parsed
				}
			}
			if projects := extractIDs(row["Related Projects"]); len(projects) > 0 {
				c.Projects = projects
			}
			if evidence := extractIDs(row["Supporting Evidence"]); len(evidence) > 0 {
				c.EvidenceObjectIDs = evidence
			}
			claims = append(claims, c)
		}

		bullets := parseBulletList(sec.Body)
		for _, bullet := range bullets {
			normalizedSkill := strings.ToLower(strings.Join(strings.Fields(bullet), " "))
			if seenSkills[normalizedSkill] {
				continue
			}
			seenSkills[normalizedSkill] = true
			c := buildBaseClaim(obj, KindTechnicalSkill, bullet, ClaimSubject(obj.ID), "skilled_in", ClaimValue(bullet), nil)
			c.Skills = []string{bullet}
			claims = append(claims, c)
		}
	}
	return claims
}

func extractFromAccomplishment(obj *model.Object) []Claim {
	var claims []Claim
	for _, sec := range obj.Sections {
		bullets := parseBulletList(sec.Body)
		for _, bullet := range bullets {
			claims = append(claims, buildBaseClaim(obj, KindAccomplishment, bullet, ClaimSubject(obj.ID), "accomplished", ClaimValue(bullet), nil))
		}
	}
	return claims
}

func extractExperienceOrganization(kvs map[string]string) string {
	for _, key := range []string{"Organization", "Employer", "Company"} {
		if org := strings.TrimSpace(kvs[key]); org != "" {
			return org
		}
	}
	return ""
}

func addOrganization(c *Claim, organization string) {
	if organization != "" {
		c.Organizations = []string{organization}
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func buildBaseClaim(obj *model.Object, kind ClaimKind, stmt string, sub ClaimSubject, pred ClaimPredicate, val ClaimValue, tr *TimeRange) Claim {
	return Claim{
		ID:                GenerateClaimID([]string{obj.ID}, kind, stmt),
		Kind:              kind,
		Statement:         stmt,
		Subject:           sub,
		Predicate:         pred,
		Value:             val,
		SourceObjectIDs:   []string{obj.ID},
		EvidenceObjectIDs: obj.Metadata.RelatedEvs,
		Verification:      obj.Metadata.Verification,
		Confidence:        obj.Metadata.Confidence,
		Visibility:        obj.Metadata.Visibility,
		TimeRange:         tr,
		SourceLocations:   []model.SourceLocation{{FilePath: obj.SourceFile, Line: 1}},
		Status:            string(obj.Metadata.Status),
		Lifecycle:         string(obj.Metadata.Lifecycle),
	}
}

// GenerateClaimID creates a deterministic stable unique identifier for a claim.
func GenerateClaimID(sourceIDs []string, kind ClaimKind, statement string) ClaimID {
	sortedSources := make([]string, len(sourceIDs))
	copy(sortedSources, sourceIDs)
	sort.Strings(sortedSources)

	normalized := strings.ToLower(strings.Join(strings.Fields(statement), " "))
	combined := strings.Join(sortedSources, ",") + "|" + string(kind) + "|" + normalized

	hash := sha256.Sum256([]byte(combined))
	hashHex := hex.EncodeToString(hash[:])

	prefix := "claim"
	if len(sortedSources) > 0 {
		cleanSource := strings.ReplaceAll(sortedSources[0], ":", "-")
		prefix = fmt.Sprintf("claim:%s", cleanSource)
	}
	return ClaimID(fmt.Sprintf("%s:%s:%s", prefix, kind, hashHex[:12]))
}

func parseKeyValues(body string) map[string]string {
	kvs := make(map[string]string)
	lines := strings.Split(body, "\n")
	for _, line := range lines {
		trimmed, _ := stripMarkdownListMarker(line)
		if strings.Contains(trimmed, ":") {
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 {
				k := strings.Trim(parts[0], "* \t")
				v := strings.TrimSpace(parts[1])
				kvs[k] = v
			}
		}
	}
	return kvs
}

func parseBulletList(body string) []string {
	var items []string
	lines := strings.Split(body, "\n")
	for _, line := range lines {
		trimmed, isBullet := stripMarkdownListMarker(line)
		if trimmed == "" {
			continue
		}
		if isBullet {
			items = append(items, trimmed)
		} else if len(trimmed) > 2 && trimmed[1] == '.' {
			item := strings.TrimSpace(trimmed[2:])
			if item != "" {
				items = append(items, item)
			}
		}
	}
	return items
}

func parseSkillTableRows(body string) []map[string]string {
	var rows []map[string]string
	for _, table := range parseMarkdownTables(body) {
		headerMap := make(map[string]int)
		for idx, header := range table.headers {
			headerMap[canonicalTableHeader(header)] = idx
		}
		skillNameIdx, hasSkillName := headerMap["Skill Name"]
		if !hasSkillName {
			continue
		}
		for _, cells := range table.rows {
			if skillNameIdx >= len(cells) {
				continue
			}
			row := make(map[string]string)
			for canonical, idx := range headerMap {
				if idx < len(cells) {
					row[canonical] = cleanMarkdownCell(cells[idx])
				}
			}
			if strings.TrimSpace(row["Skill Name"]) != "" {
				rows = append(rows, row)
			}
		}
	}
	return rows
}

type markdownTable struct {
	headers []string
	rows    [][]string
}

func parseMarkdownTables(body string) []markdownTable {
	var tables []markdownTable
	lines := strings.Split(body, "\n")
	for idx := 0; idx < len(lines); {
		line := strings.TrimSpace(lines[idx])
		if !isMarkdownTableLine(line) {
			idx++
			continue
		}

		var tableLines []string
		for idx < len(lines) && isMarkdownTableLine(strings.TrimSpace(lines[idx])) {
			tableLines = append(tableLines, strings.TrimSpace(lines[idx]))
			idx++
		}
		if len(tableLines) < 2 {
			continue
		}

		headers := splitMarkdownTableRow(tableLines[0])
		var rows [][]string
		for _, rowLine := range tableLines[1:] {
			cells := splitMarkdownTableRow(rowLine)
			if isMarkdownSeparatorRow(cells) {
				continue
			}
			rows = append(rows, cells)
		}
		tables = append(tables, markdownTable{headers: headers, rows: rows})
	}
	return tables
}

func isMarkdownTableLine(line string) bool {
	return strings.HasPrefix(line, "|") && strings.Count(line, "|") >= 2
}

func splitMarkdownTableRow(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")
	parts := strings.Split(line, "|")
	cells := make([]string, 0, len(parts))
	for _, part := range parts {
		cells = append(cells, strings.TrimSpace(part))
	}
	return cells
}

func isMarkdownSeparatorRow(cells []string) bool {
	if len(cells) == 0 {
		return false
	}
	for _, cell := range cells {
		cell = strings.TrimSpace(cell)
		if cell == "" {
			return false
		}
		for _, r := range cell {
			if r != ':' && r != '-' {
				return false
			}
		}
	}
	return true
}

func canonicalTableHeader(header string) string {
	return strings.Join(strings.Fields(cleanMarkdownCell(header)), " ")
}

func cleanMarkdownCell(cell string) string {
	cell = strings.TrimSpace(cell)
	cell = strings.Trim(cell, "`")
	cell = strings.ReplaceAll(cell, "**", "")
	cell = strings.ReplaceAll(cell, "__", "")
	return strings.TrimSpace(cell)
}

func extractIDs(text string) []string {
	if strings.TrimSpace(text) == "" || strings.EqualFold(strings.TrimSpace(text), "None") {
		return nil
	}
	seen := make(map[string]bool)
	for _, match := range idReferenceRegex.FindAllString(text, -1) {
		seen[match] = true
	}
	ids := make([]string, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func stripMarkdownListMarker(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if len(trimmed) >= 2 {
		switch trimmed[0] {
		case '-', '*', '+':
			if trimmed[1] == ' ' || trimmed[1] == '\t' {
				return strings.TrimSpace(trimmed[1:]), true
			}
		}
	}
	return trimmed, false
}

func parseMetrics(statement string) []Metric {
	var metrics []Metric
	stmtLower := strings.ToLower(statement)

	// Lexically search for percent values
	rePct := regexp.MustCompile(`(\d+(?:\.\d+)?)\s*(?:%|percent)`)
	pctMatches := rePct.FindAllStringSubmatch(statement, -1)
	for _, m := range pctMatches {
		val, _ := strconv.ParseFloat(m[1], 64)
		metric := Metric{
			Name:   "Percentage Outperformance",
			Value:  val,
			Unit:   "%",
			Status: "Approved",
		}
		classifyMetricSemantic(stmtLower, &metric)
		metrics = append(metrics, metric)
	}

	// Lexically search for throughput
	reRps := regexp.MustCompile(`(\d{1,3}(?:,\d{3})*(?:\.\d+)?)\s*(?:requests per second|rps)`)
	rpsMatches := reRps.FindAllStringSubmatch(statement, -1)
	for _, m := range rpsMatches {
		cleanNum := strings.ReplaceAll(m[1], ",", "")
		val, _ := strconv.ParseFloat(cleanNum, 64)
		metric := Metric{
			Name:   "Throughput",
			Value:  val,
			Unit:   "RPS",
			Status: "Approved",
		}
		classifyMetricSemantic(stmtLower, &metric)
		metrics = append(metrics, metric)
	}

	return metrics
}

func classifyMetricSemantic(stmtLower string, m *Metric) {
	// 1. Version / Identifier Check (e.g. version 2.0)
	if strings.Contains(stmtLower, "version") || strings.Contains(stmtLower, " v") {
		m.Status = "Ambiguous"
		m.IsAmbiguous = true
		m.Reason = "Contains version identifier context"
		return
	}

	// 2. Target / Goal Check (e.g. target was 99.9%)
	if strings.Contains(stmtLower, "target") || strings.Contains(stmtLower, "goal") {
		m.Status = "Target"
		m.IsAmbiguous = true
		m.Reason = "Identified as target or goal rather than completed result"
		return
	}

	// 3. Estimate / Forecast / Projection Check (e.g. could save 20 hours, expected 30%)
	if strings.Contains(stmtLower, "could") || strings.Contains(stmtLower, "expected") ||
		strings.Contains(stmtLower, "estimate") || strings.Contains(stmtLower, "forecast") ||
		strings.Contains(stmtLower, "projection") || strings.Contains(stmtLower, "potential") ||
		strings.Contains(stmtLower, "planned") {
		m.Status = "Estimate"
		m.IsAmbiguous = true
		m.Reason = "Identified as estimate, forecast, or potential projection"
		return
	}

	// 4. Negative result or no improvement check
	if strings.Contains(stmtLower, "no measurable") || strings.Contains(stmtLower, "did not") ||
		strings.Contains(stmtLower, "no improvement") || strings.Contains(stmtLower, "without improvement") ||
		strings.Contains(stmtLower, "failure rate") {
		m.Status = "Ambiguous"
		m.IsAmbiguous = true
		m.Reason = "Identified as negative outcome or contextual warning"
		return
	}

	// 5. Contextual Number/Team size/Budget/Duration
	if strings.Contains(stmtLower, " team") || strings.Contains(stmtLower, " budget") ||
		strings.Contains(stmtLower, " months") || strings.Contains(stmtLower, " weeks") ||
		strings.Contains(stmtLower, " years") || strings.Contains(stmtLower, " technician") ||
		strings.Contains(stmtLower, " supporting ") || strings.Contains(stmtLower, " application") {
		m.Status = "Contextual"
		m.IsAmbiguous = true
		m.Reason = "Identified as team size, budget, duration, or capacity context"
		return
	}

	// 6. Approximate Range check
	if strings.Contains(stmtLower, "approximately") || strings.Contains(stmtLower, "approx") ||
		strings.Contains(stmtLower, "about") || strings.Contains(stmtLower, "around") {
		m.Status = "Ambiguous"
		m.IsAmbiguous = true
		m.Reason = "Identified as approximate or range metric value"
		return
	}

	// 7. Causal / Outperformance validation
	hasOutperformanceVerb := false
	verbs := []string{"reduce", "cut", "save", "increase", "improve", "optimize", "grow", "lower", "scale", "boost", "decrease"}
	for _, v := range verbs {
		if strings.Contains(stmtLower, v) {
			hasOutperformanceVerb = true
			break
		}
	}
	if !hasOutperformanceVerb {
		m.Status = "Ambiguous"
		m.IsAmbiguous = true
		m.Reason = "Lacks explicit outperformance action verb context"
	}
}

func parseTimeRange(durationStr string) *TimeRange {
	durationStr = strings.ReplaceAll(durationStr, "–", "-")
	durationStr = strings.ReplaceAll(durationStr, "—", "-")

	// Try splitting by space-hyphen-space first
	parts := strings.Split(durationStr, " - ")
	if len(parts) != 2 {
		parts = strings.Split(strings.ToLower(durationStr), " to ")
	}
	if len(parts) != 2 {
		// Fallback for simple hyphen split
		parts = strings.Split(durationStr, "-")
		if len(parts) == 4 { // e.g. 2024-01-2025-01
			parts = []string{parts[0] + "-" + parts[1], parts[2] + "-" + parts[3]}
		}
	}
	if len(parts) != 2 {
		return nil
	}
	startStr := strings.TrimSpace(parts[0])
	endStr := strings.TrimSpace(parts[1])

	start, err := time.Parse("2006-01", startStr)
	if err != nil {
		start, err = time.Parse("2006", startStr)
		if err != nil {
			return nil
		}
	}

	var end time.Time
	if strings.ToLower(endStr) != "present" && strings.ToLower(endStr) != "current" && endStr != "" {
		end, err = time.Parse("2006-01", endStr)
		if err != nil {
			end, err = time.Parse("2006", endStr)
			if err != nil {
				return &TimeRange{Start: start}
			}
		}
	}

	return &TimeRange{Start: start, End: end}
}
