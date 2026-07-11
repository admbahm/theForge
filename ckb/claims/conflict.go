package claims

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"
)

// ClaimConflict records validation issues between incompatible claims.
type ClaimConflict struct {
	ID              string   `json:"id"`
	Type            string   `json:"type"`     // "DateOverlap", "MetricMismatch", "TitleMismatch"
	Severity        string   `json:"severity"` // "Error", "Warning"
	Explanation     string   `json:"explanation"`
	ClaimIDs        []string `json:"claim_ids"`
	SourceObjectIDs []string `json:"source_object_ids"`
	Blocked         bool     `json:"blocked"`
}

// DetectConflicts scans a slice of claims and returns all detected conflicts.
func DetectConflicts(claims []Claim) []ClaimConflict {
	var conflicts []ClaimConflict

	// Sort claims by ID first for deterministic pairing
	sortedClaims := make([]Claim, len(claims))
	copy(sortedClaims, claims)
	sort.Slice(sortedClaims, func(i, j int) bool {
		return sortedClaims[i].ID < sortedClaims[j].ID
	})

	for i := 0; i < len(sortedClaims); i++ {
		for j := i + 1; j < len(sortedClaims); j++ {
			c1 := sortedClaims[i]
			c2 := sortedClaims[j]

			// 1. Job Title Mismatch for the same organization at the same time
			if c1.Kind == KindRole && c2.Kind == KindRole {
				// Same organization
				if len(c1.SourceObjectIDs) > 0 && len(c2.SourceObjectIDs) > 0 && c1.SourceObjectIDs[0] == c2.SourceObjectIDs[0] {
					if c1.Value != c2.Value && isTimeRangeOverlap(c1.TimeRange, c2.TimeRange) {
						explanation := fmt.Sprintf("Job Title Mismatch: Job titles %q and %q conflict during overlapping period on source %s", c1.Value, c2.Value, c1.SourceObjectIDs[0])
						conflicts = append(conflicts, buildConflict("TitleMismatch", "Error", explanation, c1, c2, true))
					}
				}
			}

			// 2. Metric Mismatch for the same comparable metric identity under same source object
			if c1.Kind == KindMeasurableResult && c2.Kind == KindMeasurableResult {
				if len(c1.SourceObjectIDs) > 0 && len(c2.SourceObjectIDs) > 0 && c1.SourceObjectIDs[0] == c2.SourceObjectIDs[0] {
					for _, m1 := range c1.Metrics {
						for _, m2 := range c2.Metrics {
							if metricsComparableForConflict(m1, m2) && m1.Value != m2.Value {
								explanation := fmt.Sprintf("Metric Mismatch: Metric %q has conflicting values (%.2f vs %.2f) under %s", m1.Name, m1.Value, m2.Value, c1.SourceObjectIDs[0])
								conflicts = append(conflicts, buildConflict("MetricMismatch", "Error", explanation, c1, c2, true))
							}
						}
					}
				}
			}

			// 3. Overlapping Full-Time Employment
			if c1.Kind == KindEmployment && c2.Kind == KindEmployment {
				if len(c1.SourceObjectIDs) > 0 && len(c2.SourceObjectIDs) > 0 && c1.SourceObjectIDs[0] != c2.SourceObjectIDs[0] {
					if isTimeRangeOverlap(c1.TimeRange, c2.TimeRange) {
						explanation := fmt.Sprintf("Overlapping Employment: Simultaneous employment detected at %s and %s", c1.Value, c2.Value)
						conflicts = append(conflicts, buildConflict("DateOverlap", "Warning", explanation, c1, c2, false))
					}
				}
			}
		}
	}

	return conflicts
}

func metricsComparableForConflict(m1, m2 Metric) bool {
	if m1.Name == "" || m2.Name == "" || m1.Unit == "" || m2.Unit == "" {
		return false
	}
	if m1.Name != m2.Name || m1.Unit != m2.Unit {
		return false
	}
	if m1.Name == "Percentage Outperformance" && m1.Unit == "%" {
		return false
	}
	return true
}

func buildConflict(conflictType, severity, explanation string, c1, c2 Claim, blocked bool) ClaimConflict {
	claimIDs := []string{string(c1.ID), string(c2.ID)}
	sourceIDs := append(c1.SourceObjectIDs, c2.SourceObjectIDs...)

	// Deduplicate source IDs
	uniqSources := make(map[string]bool)
	var cleanSources []string
	for _, s := range sourceIDs {
		if !uniqSources[s] {
			uniqSources[s] = true
			cleanSources = append(cleanSources, s)
		}
	}
	sort.Strings(cleanSources)

	return ClaimConflict{
		ID:              generateConflictID(claimIDs, conflictType),
		Type:            conflictType,
		Severity:        severity,
		Explanation:     explanation,
		ClaimIDs:        claimIDs,
		SourceObjectIDs: cleanSources,
		Blocked:         blocked,
	}
}

func generateConflictID(claimIDs []string, conflictType string) string {
	sorted := make([]string, len(claimIDs))
	copy(sorted, claimIDs)
	sort.Strings(sorted)
	combined := strings.Join(sorted, ",") + "|" + conflictType
	hash := sha256.Sum256([]byte(combined))
	return fmt.Sprintf("conflict:%s", hex.EncodeToString(hash[:])[:12])
}

func isTimeRangeOverlap(tr1, tr2 *TimeRange) bool {
	if tr1 == nil || tr2 == nil {
		return false
	}
	end1 := tr1.End
	if end1.IsZero() {
		end1 = time.Now()
	}
	end2 := tr2.End
	if end2.IsZero() {
		end2 = time.Now()
	}
	return tr1.Start.Before(end2) && tr2.Start.Before(end1)
}
