/*
Copyright 2026 The Korp Authors.

Licensed under the MIT License.
*/

package reporter

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	client   kubernetes.Interface
}

// NewEventReporter creates a new EventReporter instance
func NewEventReporter(client kubernetes.Interface, scheme *runtime.Scheme) *EventReporter {
	broadcaster := record.NewBroadcaster()
	broadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{
		Interface: client.CoreV1().Events(""),
	})
	recorder := broadcaster.NewRecorder(scheme, corev1.EventSource{Component: "korp"})

	return &EventReporter{recorder: recorder, client: client}
}

// CreateEvents creates Kubernetes events for each finding (attached to the orphaned resource) and a summary event
func (r *EventReporter) CreateEvents(ctx context.Context, korpScan *korpv1alpha1.KorpScan, result *scan.ScanResult) {
	// Determine event severity
	severity := korpScan.Spec.Reporting.EventSeverity
	if severity == "" {
		severity = "Warning"
	}

	// Create events for individual findings attached to the actual orphaned resources
	// This avoids event aggregation since each event has a different involvedObject
	for _, finding := range result.Details {
		obj := r.getResourceObject(ctx, finding)
		if obj != nil {
			reason := "Orphaned"
			message := fmt.Sprintf("Resource is orphaned (%s) - detected by korp", finding.Reason)
			r.recorder.Event(obj, severity, reason, message)
		}
	}

	// Create summary event on KorpScan
	totalOrphans := result.Summary.TotalOrphans()
	summary := buildSummaryMessage(totalOrphans, &result.Summary)
	r.recorder.Event(korpScan, "Normal", "ScanCompleted", summary)
}

// getResourceObject fetches the actual Kubernetes resource object for a finding
func (r *EventReporter) getResourceObject(ctx context.Context, finding korpv1alpha1.Finding) runtime.Object {
	switch finding.ResourceType {
	case "ConfigMap":
		obj, err := r.client.CoreV1().ConfigMaps(finding.Namespace).Get(ctx, finding.Name, metav1.GetOptions{})
		if err == nil {
			return obj
		}
	case "Secret":
		obj, err := r.client.CoreV1().Secrets(finding.Namespace).Get(ctx, finding.Name, metav1.GetOptions{})
		if err == nil {
			return obj
		}
	case "PersistentVolumeClaim":
		obj, err := r.client.CoreV1().PersistentVolumeClaims(finding.Namespace).Get(ctx, finding.Name, metav1.GetOptions{})
		if err == nil {
			return obj
		}
	case "Service":
		obj, err := r.client.CoreV1().Services(finding.Namespace).Get(ctx, finding.Name, metav1.GetOptions{})
		if err == nil {
			return obj
		}
	case "ServiceAccount":
		obj, err := r.client.CoreV1().ServiceAccounts(finding.Namespace).Get(ctx, finding.Name, metav1.GetOptions{})
		if err == nil {
			return obj
		}
	case "Deployment":
		obj, err := r.client.AppsV1().Deployments(finding.Namespace).Get(ctx, finding.Name, metav1.GetOptions{})
		if err == nil {
			return obj
		}
	case "StatefulSet":
		obj, err := r.client.AppsV1().StatefulSets(finding.Namespace).Get(ctx, finding.Name, metav1.GetOptions{})
		if err == nil {
			return obj
		}
	case "DaemonSet":
		obj, err := r.client.AppsV1().DaemonSets(finding.Namespace).Get(ctx, finding.Name, metav1.GetOptions{})
		if err == nil {
			return obj
		}
	case "ReplicaSet":
		obj, err := r.client.AppsV1().ReplicaSets(finding.Namespace).Get(ctx, finding.Name, metav1.GetOptions{})
		if err == nil {
			return obj
		}
	case "Job":
		obj, err := r.client.BatchV1().Jobs(finding.Namespace).Get(ctx, finding.Name, metav1.GetOptions{})
		if err == nil {
			return obj
		}
	case "CronJob":
		obj, err := r.client.BatchV1().CronJobs(finding.Namespace).Get(ctx, finding.Name, metav1.GetOptions{})
		if err == nil {
			return obj
		}
	case "Ingress":
		obj, err := r.client.NetworkingV1().Ingresses(finding.Namespace).Get(ctx, finding.Name, metav1.GetOptions{})
		if err == nil {
			return obj
		}
	case "Role":
		obj, err := r.client.RbacV1().Roles(finding.Namespace).Get(ctx, finding.Name, metav1.GetOptions{})
		if err == nil {
			return obj
		}
	case "ClusterRole":
		obj, err := r.client.RbacV1().ClusterRoles().Get(ctx, finding.Name, metav1.GetOptions{})
		if err == nil {
			return obj
		}
	case "RoleBinding":
		obj, err := r.client.RbacV1().RoleBindings(finding.Namespace).Get(ctx, finding.Name, metav1.GetOptions{})
		if err == nil {
			return obj
		}
	case "ClusterRoleBinding":
		obj, err := r.client.RbacV1().ClusterRoleBindings().Get(ctx, finding.Name, metav1.GetOptions{})
		if err == nil {
			return obj
		}
	}
	return nil
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
		{"Roles", summary.OrphanedRoles},
		{"ClusterRoles", summary.OrphanedClusterRoles},
		{"RoleBindings", summary.OrphanedRoleBindings},
		{"ClusterRoleBindings", summary.OrphanedClusterRoleBindings},
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
