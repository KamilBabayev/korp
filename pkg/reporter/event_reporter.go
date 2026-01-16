/*
Copyright 2026 The Korp Authors.

Licensed under the MIT License.
*/

package reporter

import (
	"context"
	"fmt"

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

	// Create events for individual findings
	for _, finding := range result.Details {
		message := fmt.Sprintf("Orphaned %s detected: %s/%s (%s)",
			finding.ResourceType, finding.Namespace, finding.Name, finding.Reason)

		r.recorder.Event(korpScan, severity, "OrphanDetected", message)
	}

	// Create summary event
	totalOrphans := result.Summary.TotalOrphans()
	summary := fmt.Sprintf("Scan completed: found %d orphaned resources (CM:%d, Sec:%d, PVC:%d, Dep:%d, Job:%d, Ing:%d, Svc:%d, STS:%d, DS:%d, CJ:%d, RS:%d, SA:%d)",
		totalOrphans,
		result.Summary.OrphanedConfigMaps,
		result.Summary.OrphanedSecrets,
		result.Summary.OrphanedPVCs,
		result.Summary.OrphanedDeployments,
		result.Summary.OrphanedJobs,
		result.Summary.OrphanedIngresses,
		result.Summary.ServicesWithoutEndpoints,
		result.Summary.OrphanedStatefulSets,
		result.Summary.OrphanedDaemonSets,
		result.Summary.OrphanedCronJobs,
		result.Summary.OrphanedReplicaSets,
		result.Summary.OrphanedServiceAccounts)

	r.recorder.Event(korpScan, "Normal", "ScanCompleted", summary)
}

// CreateEvent creates a single Kubernetes event
func (r *EventReporter) CreateEvent(obj runtime.Object, eventType, reason, message string) {
	r.recorder.Event(obj, eventType, reason, message)
}
