package claims

import (
	"testing"

	"github.com/admbahm/theForge/ckb/model"
)

func TestClaimStrengthPreservation(t *testing.T) {
	// Verify that approximate metrics are flagged as ambiguous to prevent strength upgrades
	approxBullet := "Improved performance by approximately 15%"
	metrics := parseMetrics(approxBullet)
	if len(metrics) == 0 {
		t.Fatalf("Expected 1 metric extracted, got 0")
	}
	if !metrics[0].IsAmbiguous || metrics[0].Status != "Ambiguous" {
		t.Errorf("Expected approximate metric to be flagged as Ambiguous, got Status=%q, IsAmbiguous=%t", metrics[0].Status, metrics[0].IsAmbiguous)
	}

	// Verify that target goals are flagged as Target
	targetBullet := "Target was 99.9% availability"
	metrics = parseMetrics(targetBullet)
	if len(metrics) == 0 {
		t.Fatalf("Expected 1 metric extracted, got 0")
	}
	if !metrics[0].IsAmbiguous || metrics[0].Status != "Target" {
		t.Errorf("Expected target metric to be flagged as Target, got Status=%q, IsAmbiguous=%t", metrics[0].Status, metrics[0].IsAmbiguous)
	}

	// Verify that estimates are flagged as Estimate
	estimateBullet := "Expected improvement of 30%"
	metrics = parseMetrics(estimateBullet)
	if len(metrics) == 0 {
		t.Fatalf("Expected 1 metric extracted, got 0")
	}
	if !metrics[0].IsAmbiguous || metrics[0].Status != "Estimate" {
		t.Errorf("Expected expected metric to be flagged as Estimate, got Status=%q, IsAmbiguous=%t", metrics[0].Status, metrics[0].IsAmbiguous)
	}

	// Verify that software versions are NOT promoted to metrics
	versionBullet := "Worked on version 2.0 of the application"
	metrics = parseMetrics(versionBullet)
	for _, m := range metrics {
		if m.Status == "Approved" && !m.IsAmbiguous {
			t.Errorf("Expected version 2.0 to be flagged as ambiguous version, got approved metric: %+v", m)
		}
	}

	// Verify that team context / size is contextual
	teamBullet := "Responsible for a team supporting 40 applications"
	metrics = parseMetrics(teamBullet)
	for _, m := range metrics {
		if m.Status == "Approved" && !m.IsAmbiguous {
			t.Errorf("Expected team size metric to be flagged as contextual/ambiguous, got approved: %+v", m)
		}
	}

	// Verify that negative results are flagged as ambiguous
	negBullet := "Processed 1,000 requests, but did not improve throughput"
	metrics = parseMetrics(negBullet)
	for _, m := range metrics {
		if m.Status == "Approved" && !m.IsAmbiguous {
			t.Errorf("Expected negative result to be flagged as ambiguous, got approved: %+v", m)
		}
	}

	// Verify that valid outperformance is approved
	validBullet := "Reduced latency from 400 ms to 250 ms and cut error rate by 40%"
	metrics = parseMetrics(validBullet)
	if len(metrics) == 0 {
		t.Fatalf("Expected metrics extracted, got 0")
	}
	for _, m := range metrics {
		if m.IsAmbiguous || m.Status != "Approved" {
			t.Errorf("Expected valid outperformance to be approved, got Status=%q, IsAmbiguous=%t", m.Status, m.IsAmbiguous)
		}
	}
}

func TestGenericPercentageMetricsDoNotBlockAsConflicts(t *testing.T) {
	claims := []Claim{
		measurableClaimForConflict("claim:utilization", "proj:titan-like", "Raised utilization from 15% to 62%.", []Metric{
			{Name: "Percentage Outperformance", Value: 15, Unit: "%", Status: "Approved"},
			{Name: "Percentage Outperformance", Value: 62, Unit: "%", Status: "Approved"},
		}),
		measurableClaimForConflict("claim:remediation", "proj:titan-like", "Completed 100% configuration remediation.", []Metric{
			{Name: "Percentage Outperformance", Value: 100, Unit: "%", Status: "Approved"},
		}),
		measurableClaimForConflict("claim:defects", "proj:titan-like", "Reduced defects by 30%.", []Metric{
			{Name: "Percentage Outperformance", Value: 30, Unit: "%", Status: "Approved"},
		}),
		measurableClaimForConflict("claim:availability", "proj:titan-like", "Maintained 99.9% availability.", []Metric{
			{Name: "Percentage Outperformance", Value: 99.9, Unit: "%", Status: "Approved"},
		}),
	}

	conflicts := DetectConflicts(claims)
	for _, conf := range conflicts {
		if conf.Type == "MetricMismatch" && conf.Blocked {
			t.Fatalf("Generic percentage metrics should not produce blocking conflicts: %+v", conf)
		}
	}
}

func TestExplicitSemanticMetricConflictsStillBlock(t *testing.T) {
	conflicts := DetectConflicts([]Claim{
		measurableClaimForConflict("claim:utilization-62", "proj:titan-like", "Recorded utilization at 62%.", []Metric{
			{Name: "Cluster Utilization", Value: 62, Unit: "%", Status: "Approved"},
		}),
		measurableClaimForConflict("claim:utilization-71", "proj:titan-like", "Recorded utilization at 71%.", []Metric{
			{Name: "Cluster Utilization", Value: 71, Unit: "%", Status: "Approved"},
		}),
	})

	if len(conflicts) != 1 {
		t.Fatalf("Expected one explicit semantic metric conflict, got %+v", conflicts)
	}
	if !conflicts[0].Blocked || conflicts[0].Type != "MetricMismatch" {
		t.Fatalf("Expected blocking metric mismatch, got %+v", conflicts[0])
	}

	duplicates := DetectConflicts([]Claim{
		measurableClaimForConflict("claim:remediation-a", "proj:titan-like", "Recorded remediation at 100%.", []Metric{
			{Name: "Configuration Remediation", Value: 100, Unit: "%", Status: "Approved"},
		}),
		measurableClaimForConflict("claim:remediation-b", "proj:titan-like", "Recorded remediation at 100%.", []Metric{
			{Name: "Configuration Remediation", Value: 100, Unit: "%", Status: "Approved"},
		}),
	})
	if len(duplicates) != 0 {
		t.Fatalf("Exact duplicate metric values should not conflict: %+v", duplicates)
	}
}

func measurableClaimForConflict(id ClaimID, sourceID string, statement string, metrics []Metric) Claim {
	return Claim{
		ID:              id,
		Kind:            KindMeasurableResult,
		Statement:       statement,
		Value:           ClaimValue(statement),
		SourceObjectIDs: []string{sourceID},
		Verification:    model.VerificationIndependentlyVerified,
		Confidence:      1.0,
		Visibility:      model.VisibilityPublic,
		Status:          "Active",
		Metrics:         metrics,
	}
}
