/*
Copyright 2026 The Korp Authors.

Licensed under the MIT License.
*/

package scan

import (
	"context"
	"regexp"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	korpv1alpha1 "github.com/kamilbabayev/korp/api/v1alpha1"
	k8sutil "github.com/kamilbabayev/korp/pkg/k8s"
)

// Scanner performs scans of Kubernetes resources for orphans
type Scanner struct {
	client *kubernetes.Clientset
}

// NewScanner creates a new Scanner instance
func NewScanner(client *kubernetes.Clientset) *Scanner {
	return &Scanner{client: client}
}

// Scan performs a scan based on the KorpScan specification
func (s *Scanner) Scan(ctx context.Context, korpScan *korpv1alpha1.KorpScan) (*ScanResult, error) {
	result := &ScanResult{}
	now := metav1.Time{Time: time.Now()}

	// Determine which resource types to scan
	types := korpScan.Spec.ResourceTypes
	if len(types) == 0 {
		// Default to all resource types
		types = []string{"configmaps", "secrets", "pvcs", "services", "deployments", "jobs", "ingresses",
			"statefulsets", "daemonsets", "cronjobs", "replicasets", "serviceaccounts"}
	}

	// Get list of namespaces to scan
	namespacesToScan, err := s.getNamespacesToScan(ctx, korpScan)
	if err != nil {
		return nil, err
	}

	// Scan each namespace
	for _, ns := range namespacesToScan {
		if err := s.scanNamespace(ctx, ns, types, korpScan, result, now); err != nil {
			return nil, err
		}
	}

	// Update total resources count
	result.Summary.TotalResources = len(result.Details)

	return result, nil
}

// getNamespacesToScan returns the list of namespaces to scan based on the KorpScan spec
func (s *Scanner) getNamespacesToScan(ctx context.Context, korpScan *korpv1alpha1.KorpScan) ([]string, error) {
	targetNs := korpScan.Spec.TargetNamespace

	// If not scanning all namespaces, return the single target
	if targetNs != "*" {
		return []string{targetNs}, nil
	}

	// Get all namespaces
	nsList, err := s.client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	// Build exclusion set
	excludeSet := make(map[string]bool)
	for _, ns := range korpScan.Spec.Filters.ExcludeNamespaces {
		excludeSet[ns] = true
	}

	// Filter namespaces
	var namespaces []string
	for _, ns := range nsList.Items {
		if !excludeSet[ns.Name] {
			namespaces = append(namespaces, ns.Name)
		}
	}

	return namespaces, nil
}

// scanNamespace scans a single namespace for orphaned resources
func (s *Scanner) scanNamespace(ctx context.Context, ns string, types []string, korpScan *korpv1alpha1.KorpScan, result *ScanResult, now metav1.Time) error {
	// Scan each requested resource type
	for _, rt := range types {
		switch rt {
		case "configmaps":
			if err := s.scanConfigMaps(ctx, ns, korpScan, result, now); err != nil {
				return err
			}

		case "secrets":
			if err := s.scanSecrets(ctx, ns, korpScan, result, now); err != nil {
				return err
			}

		case "pvcs":
			if err := s.scanPVCs(ctx, ns, korpScan, result, now); err != nil {
				return err
			}

		case "services":
			if err := s.scanServices(ctx, ns, korpScan, result, now); err != nil {
				return err
			}

		case "deployments":
			if err := s.scanDeployments(ctx, ns, korpScan, result, now); err != nil {
				return err
			}

		case "jobs":
			if err := s.scanJobs(ctx, ns, korpScan, result, now); err != nil {
				return err
			}

		case "ingresses":
			if err := s.scanIngresses(ctx, ns, korpScan, result, now); err != nil {
				return err
			}

		case "statefulsets":
			if err := s.scanStatefulSets(ctx, ns, korpScan, result, now); err != nil {
				return err
			}

		case "daemonsets":
			if err := s.scanDaemonSets(ctx, ns, korpScan, result, now); err != nil {
				return err
			}

		case "cronjobs":
			if err := s.scanCronJobs(ctx, ns, korpScan, result, now); err != nil {
				return err
			}

		case "replicasets":
			if err := s.scanReplicaSets(ctx, ns, korpScan, result, now); err != nil {
				return err
			}

		case "serviceaccounts":
			if err := s.scanServiceAccounts(ctx, ns, korpScan, result, now); err != nil {
				return err
			}
		}
	}

	return nil
}

// scanConfigMaps scans for orphaned ConfigMaps
func (s *Scanner) scanConfigMaps(ctx context.Context, ns string, korpScan *korpv1alpha1.KorpScan, result *ScanResult, detectedAt metav1.Time) error {
	orphans, err := k8sutil.OrphanConfigMaps(ctx, s.client, ns)
	if err != nil {
		return err
	}

	filtered := s.applyFilters(orphans, korpScan.Spec.Filters)
	result.Summary.OrphanedConfigMaps += len(filtered)

	for _, name := range filtered {
		result.Details = append(result.Details, korpv1alpha1.Finding{
			ResourceType: "ConfigMap",
			Namespace:    ns,
			Name:         name,
			Reason:       "NoOwnerReference",
			DetectedAt:   detectedAt,
		})
	}

	return nil
}

// scanSecrets scans for orphaned Secrets
func (s *Scanner) scanSecrets(ctx context.Context, ns string, korpScan *korpv1alpha1.KorpScan, result *ScanResult, detectedAt metav1.Time) error {
	orphans, err := k8sutil.OrphanSecrets(ctx, s.client, ns)
	if err != nil {
		return err
	}

	filtered := s.applyFilters(orphans, korpScan.Spec.Filters)
	result.Summary.OrphanedSecrets += len(filtered)

	for _, name := range filtered {
		result.Details = append(result.Details, korpv1alpha1.Finding{
			ResourceType: "Secret",
			Namespace:    ns,
			Name:         name,
			Reason:       "NoOwnerReference",
			DetectedAt:   detectedAt,
		})
	}

	return nil
}

// scanPVCs scans for orphaned PersistentVolumeClaims
func (s *Scanner) scanPVCs(ctx context.Context, ns string, korpScan *korpv1alpha1.KorpScan, result *ScanResult, detectedAt metav1.Time) error {
	orphans, err := k8sutil.OrphanPVCs(ctx, s.client, ns)
	if err != nil {
		return err
	}

	filtered := s.applyFilters(orphans, korpScan.Spec.Filters)
	result.Summary.OrphanedPVCs += len(filtered)

	for _, name := range filtered {
		result.Details = append(result.Details, korpv1alpha1.Finding{
			ResourceType: "PersistentVolumeClaim",
			Namespace:    ns,
			Name:         name,
			Reason:       "NoOwnerReference",
			DetectedAt:   detectedAt,
		})
	}

	return nil
}

// scanServices scans for Services without Endpoints
func (s *Scanner) scanServices(ctx context.Context, ns string, korpScan *korpv1alpha1.KorpScan, result *ScanResult, detectedAt metav1.Time) error {
	orphans, err := k8sutil.ServicesWithoutEndpoints(ctx, s.client, ns)
	if err != nil {
		return err
	}

	filtered := s.applyFilters(orphans, korpScan.Spec.Filters)
	result.Summary.ServicesWithoutEndpoints += len(filtered)

	for _, name := range filtered {
		result.Details = append(result.Details, korpv1alpha1.Finding{
			ResourceType: "Service",
			Namespace:    ns,
			Name:         name,
			Reason:       "NoEndpoints",
			DetectedAt:   detectedAt,
		})
	}

	return nil
}

// scanDeployments scans for orphaned Deployments
func (s *Scanner) scanDeployments(ctx context.Context, ns string, korpScan *korpv1alpha1.KorpScan, result *ScanResult, detectedAt metav1.Time) error {
	orphans, err := k8sutil.OrphanDeployments(ctx, s.client, ns)
	if err != nil {
		return err
	}

	filtered := s.applyFilters(orphans, korpScan.Spec.Filters)
	result.Summary.OrphanedDeployments += len(filtered)

	for _, name := range filtered {
		result.Details = append(result.Details, korpv1alpha1.Finding{
			ResourceType: "Deployment",
			Namespace:    ns,
			Name:         name,
			Reason:       "ScaledToZero",
			DetectedAt:   detectedAt,
		})
	}

	return nil
}

// scanJobs scans for orphaned Jobs
func (s *Scanner) scanJobs(ctx context.Context, ns string, korpScan *korpv1alpha1.KorpScan, result *ScanResult, detectedAt metav1.Time) error {
	orphans, err := k8sutil.OrphanJobs(ctx, s.client, ns)
	if err != nil {
		return err
	}

	filtered := s.applyFilters(orphans, korpScan.Spec.Filters)
	result.Summary.OrphanedJobs += len(filtered)

	for _, name := range filtered {
		result.Details = append(result.Details, korpv1alpha1.Finding{
			ResourceType: "Job",
			Namespace:    ns,
			Name:         name,
			Reason:       "CompletedOld",
			DetectedAt:   detectedAt,
		})
	}

	return nil
}

// scanIngresses scans for orphaned Ingresses
func (s *Scanner) scanIngresses(ctx context.Context, ns string, korpScan *korpv1alpha1.KorpScan, result *ScanResult, detectedAt metav1.Time) error {
	orphans, err := k8sutil.OrphanIngresses(ctx, s.client, ns)
	if err != nil {
		return err
	}

	filtered := s.applyFilters(orphans, korpScan.Spec.Filters)
	result.Summary.OrphanedIngresses += len(filtered)

	for _, name := range filtered {
		result.Details = append(result.Details, korpv1alpha1.Finding{
			ResourceType: "Ingress",
			Namespace:    ns,
			Name:         name,
			Reason:       "NoBackendService",
			DetectedAt:   detectedAt,
		})
	}

	return nil
}

// scanStatefulSets scans for orphaned StatefulSets
func (s *Scanner) scanStatefulSets(ctx context.Context, ns string, korpScan *korpv1alpha1.KorpScan, result *ScanResult, detectedAt metav1.Time) error {
	orphans, err := k8sutil.OrphanStatefulSets(ctx, s.client, ns)
	if err != nil {
		return err
	}

	filtered := s.applyFilters(orphans, korpScan.Spec.Filters)
	result.Summary.OrphanedStatefulSets += len(filtered)

	for _, name := range filtered {
		result.Details = append(result.Details, korpv1alpha1.Finding{
			ResourceType: "StatefulSet",
			Namespace:    ns,
			Name:         name,
			Reason:       "ScaledToZeroOrNoReadyPods",
			DetectedAt:   detectedAt,
		})
	}

	return nil
}

// scanDaemonSets scans for orphaned DaemonSets
func (s *Scanner) scanDaemonSets(ctx context.Context, ns string, korpScan *korpv1alpha1.KorpScan, result *ScanResult, detectedAt metav1.Time) error {
	orphans, err := k8sutil.OrphanDaemonSets(ctx, s.client, ns)
	if err != nil {
		return err
	}

	filtered := s.applyFilters(orphans, korpScan.Spec.Filters)
	result.Summary.OrphanedDaemonSets += len(filtered)

	for _, name := range filtered {
		result.Details = append(result.Details, korpv1alpha1.Finding{
			ResourceType: "DaemonSet",
			Namespace:    ns,
			Name:         name,
			Reason:       "NoScheduledPods",
			DetectedAt:   detectedAt,
		})
	}

	return nil
}

// scanCronJobs scans for orphaned CronJobs
func (s *Scanner) scanCronJobs(ctx context.Context, ns string, korpScan *korpv1alpha1.KorpScan, result *ScanResult, detectedAt metav1.Time) error {
	orphans, err := k8sutil.OrphanCronJobs(ctx, s.client, ns)
	if err != nil {
		return err
	}

	filtered := s.applyFilters(orphans, korpScan.Spec.Filters)
	result.Summary.OrphanedCronJobs += len(filtered)

	for _, name := range filtered {
		result.Details = append(result.Details, korpv1alpha1.Finding{
			ResourceType: "CronJob",
			Namespace:    ns,
			Name:         name,
			Reason:       "SuspendedNoRecentSuccess",
			DetectedAt:   detectedAt,
		})
	}

	return nil
}

// scanReplicaSets scans for orphaned ReplicaSets
func (s *Scanner) scanReplicaSets(ctx context.Context, ns string, korpScan *korpv1alpha1.KorpScan, result *ScanResult, detectedAt metav1.Time) error {
	orphans, err := k8sutil.OrphanReplicaSets(ctx, s.client, ns)
	if err != nil {
		return err
	}

	filtered := s.applyFilters(orphans, korpScan.Spec.Filters)
	result.Summary.OrphanedReplicaSets += len(filtered)

	for _, name := range filtered {
		result.Details = append(result.Details, korpv1alpha1.Finding{
			ResourceType: "ReplicaSet",
			Namespace:    ns,
			Name:         name,
			Reason:       "OrphanedNoOwner",
			DetectedAt:   detectedAt,
		})
	}

	return nil
}

// scanServiceAccounts scans for orphaned ServiceAccounts
func (s *Scanner) scanServiceAccounts(ctx context.Context, ns string, korpScan *korpv1alpha1.KorpScan, result *ScanResult, detectedAt metav1.Time) error {
	orphans, err := k8sutil.OrphanServiceAccounts(ctx, s.client, ns)
	if err != nil {
		return err
	}

	filtered := s.applyFilters(orphans, korpScan.Spec.Filters)
	result.Summary.OrphanedServiceAccounts += len(filtered)

	for _, name := range filtered {
		result.Details = append(result.Details, korpv1alpha1.Finding{
			ResourceType: "ServiceAccount",
			Namespace:    ns,
			Name:         name,
			Reason:       "NotUsedByAnyPod",
			DetectedAt:   detectedAt,
		})
	}

	return nil
}

// applyFilters applies exclusion filters to a list of resource names
func (s *Scanner) applyFilters(names []string, filters korpv1alpha1.FilterSpec) []string {
	if len(filters.ExcludeNamePatterns) == 0 {
		return names
	}

	var filtered []string
	for _, name := range names {
		excluded := false

		// Check name pattern exclusions
		for _, pattern := range filters.ExcludeNamePatterns {
			matched, err := regexp.MatchString(pattern, name)
			if err != nil {
				// If regex is invalid, skip this pattern
				continue
			}
			if matched {
				excluded = true
				break
			}
		}

		if !excluded {
			filtered = append(filtered, name)
		}
	}

	return filtered
}
