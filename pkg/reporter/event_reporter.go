/*
Copyright 2026 The Korp Authors.

Licensed under the MIT License.
*/

package reporter

import (
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"

	korpv1alpha1 "github.com/kamilbabayev/korp/api/v1alpha1"
	"github.com/kamilbabayev/korp/pkg/scan"
)

// EventReporter creates Kubernetes events for scan findings
type EventReporter struct {
	recorder record.EventRecorder
}

// NewEventReporter creates a new EventReporter instance
func NewEventReporter(client kubernetes.Interface, scheme *runtime.Scheme) *EventReporter {
	broadcaster := record.NewBroadcaster()
	broadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{
		Interface: client.CoreV1().Events(""),
	})
	recorder := broadcaster.NewRecorder(scheme, corev1.EventSource{Component: "korp"})

	return &EventReporter{recorder: recorder}
}

// CreateEvents creates Kubernetes events for each finding and a summary event
func (r *EventReporter) CreateEvents(ctx context.Context, korpScan *korpv1alpha1.KorpScan, result *scan.ScanResult) {
	// Determine event severity
	severity := korpScan.Spec.Reporting.EventSeverity
	if severity == "" {
		severity = "Warning"
	}

	// Create events for individual findings with small delay to avoid rate limiting
	for i, finding := range result.Details {
		message := fmt.Sprintf("Orphaned %s detected: %s/%s (%s)",
			finding.ResourceType, finding.Namespace, finding.Name, finding.Reason)

		r.recorder.Event(korpScan, severity, "OrphanDetected", message)

		// Add small delay between events to prevent Kubernetes rate limiting
		if i < len(result.Details)-1 {
			time.Sleep(100 * time.Millisecond)
		}
	}

	// Create summary event with only non-zero counts
	totalOrphans := result.Summary.TotalOrphans()
	summary := buildSummaryMessage(totalOrphans, &result.Summary)
	r.recorder.Event(korpScan, "Normal", "ScanCompleted", summary)
}

// CreateEvent creates a single Kubernetes event
func (r *EventReporter) CreateEvent(obj runtime.Object, eventType, reason, message string) {
	r.recorder.Event(obj, eventType, reason, message)
}

// buildSummaryMessage creates a summary message showing only non-zero orphan counts
func buildSummaryMessage(totalOrphans int, summary *korpv1alpha1.ScanSummary) string {
	if totalOrphans == 0 {
		return "Scan completed: no orphaned resources found"
	}

	// Define resource types and their counts
	resourceCounts := []struct {
		name  string
		count int
	}{
		{"ConfigMaps", summary.OrphanedConfigMaps},
		{"Secrets", summary.OrphanedSecrets},
		{"PVCs", summary.OrphanedPVCs},
		{"Services", summary.ServicesWithoutEndpoints},
		{"Deployments", summary.OrphanedDeployments},
		{"StatefulSets", summary.OrphanedStatefulSets},
		{"DaemonSets", summary.OrphanedDaemonSets},
		{"Jobs", summary.OrphanedJobs},
		{"CronJobs", summary.OrphanedCronJobs},
		{"ReplicaSets", summary.OrphanedReplicaSets},
		{"Ingresses", summary.OrphanedIngresses},
		{"ServiceAccounts", summary.OrphanedServiceAccounts},
	}

	// Build list of non-zero counts
	var parts []string
	for _, rc := range resourceCounts {
		if rc.count > 0 {
			parts = append(parts, fmt.Sprintf("%s: %d", rc.name, rc.count))
		}
	}

	if len(parts) == 0 {
		return fmt.Sprintf("Scan completed: found %d orphaned resources", totalOrphans)
	}

	return fmt.Sprintf("Scan completed: found %d orphaned resources (%s)", totalOrphans, strings.Join(parts, ", "))
}
