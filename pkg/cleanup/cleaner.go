/*
Copyright 2026 The Korp Authors.

Licensed under the MIT License.
*/

package cleanup

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	korpv1alpha1 "github.com/kamilbabayev/korp/api/v1alpha1"
)

// Cleaner performs cleanup of orphaned resources
type Cleaner struct {
	client *kubernetes.Clientset
	logger logr.Logger
}

// NewCleaner creates a new Cleaner instance
func NewCleaner(client *kubernetes.Clientset, logger logr.Logger) *Cleaner {
	return &Cleaner{
		client: client,
		logger: logger,
	}
}

// CleanupResult contains the results of a cleanup operation
type CleanupResult struct {
	Summary          *korpv1alpha1.CleanupSummary
	DeletedResources []korpv1alpha1.DeletedResource
	FailedDeletions  []korpv1alpha1.FailedDeletion
}

// Clean performs cleanup based on findings and cleanup spec
func (c *Cleaner) Clean(ctx context.Context, findings []korpv1alpha1.Finding, spec *korpv1alpha1.CleanupSpec) (*CleanupResult, error) {
	result := &CleanupResult{
		Summary: &korpv1alpha1.CleanupSummary{
			DryRun: spec.IsDryRun(),
		},
	}

	if !spec.Enabled {
		c.logger.Info("Cleanup is disabled, skipping")
		return result, nil
	}

	minAge := time.Duration(spec.MinAgeDays) * 24 * time.Hour
	if spec.MinAgeDays == 0 {
		minAge = 7 * 24 * time.Hour // Default 7 days
	}

	// Build set of allowed resource types for cleanup
	allowedTypes := make(map[string]bool)
	if len(spec.ResourceTypes) > 0 {
		for _, rt := range spec.ResourceTypes {
			allowedTypes[rt] = true
		}
	}

	for _, finding := range findings {
		// Check if resource type is allowed for cleanup
		if len(allowedTypes) > 0 && !c.isResourceTypeAllowed(finding.ResourceType, allowedTypes) {
			continue
		}

		result.Summary.TotalEligible++

		// Check age threshold
		age := time.Since(finding.DetectedAt.Time)
		if age < minAge {
			result.Summary.TotalSkippedAge++
			c.logger.V(1).Info("Skipping resource due to age threshold",
				"type", finding.ResourceType,
				"namespace", finding.Namespace,
				"name", finding.Name,
				"age", age.String(),
				"minAge", minAge.String())
			continue
		}

		// Check preservation labels
		if c.hasPreservationLabel(ctx, finding, spec.PreservationLabels) {
			result.Summary.TotalSkippedPreserved++
			c.logger.Info("Skipping resource due to preservation label",
				"type", finding.ResourceType,
				"namespace", finding.Namespace,
				"name", finding.Name)
			continue
		}

		// Perform deletion (or dry-run)
		if spec.IsDryRun() {
			c.logger.Info("[DRY-RUN] Would delete resource",
				"type", finding.ResourceType,
				"namespace", finding.Namespace,
				"name", finding.Name,
				"reason", finding.Reason)
			result.Summary.TotalDeleted++
			result.DeletedResources = append(result.DeletedResources, korpv1alpha1.DeletedResource{
				ResourceType: finding.ResourceType,
				Namespace:    finding.Namespace,
				Name:         finding.Name,
				DeletedAt:    metav1.Now(),
			})
		} else {
			err := c.deleteResource(ctx, finding)
			if err != nil {
				c.logger.Error(err, "Failed to delete resource",
					"type", finding.ResourceType,
					"namespace", finding.Namespace,
					"name", finding.Name)
				result.Summary.TotalFailed++
				result.FailedDeletions = append(result.FailedDeletions, korpv1alpha1.FailedDeletion{
					ResourceType: finding.ResourceType,
					Namespace:    finding.Namespace,
					Name:         finding.Name,
					Error:        err.Error(),
				})
			} else {
				c.logger.Info("Deleted resource",
					"type", finding.ResourceType,
					"namespace", finding.Namespace,
					"name", finding.Name)
				result.Summary.TotalDeleted++
				result.DeletedResources = append(result.DeletedResources, korpv1alpha1.DeletedResource{
					ResourceType: finding.ResourceType,
					Namespace:    finding.Namespace,
					Name:         finding.Name,
					DeletedAt:    metav1.Now(),
				})
			}
		}
	}

	return result, nil
}

// isResourceTypeAllowed checks if a resource type is in the allowed list
func (c *Cleaner) isResourceTypeAllowed(resourceType string, allowedTypes map[string]bool) bool {
	// Map Finding.ResourceType to spec resource type names
	typeMapping := map[string]string{
		"ConfigMap":             "configmaps",
		"Secret":                "secrets",
		"PersistentVolumeClaim": "pvcs",
		"Service":               "services",
		"Deployment":            "deployments",
		"StatefulSet":           "statefulsets",
		"DaemonSet":             "daemonsets",
		"Job":                   "jobs",
		"CronJob":               "cronjobs",
		"ReplicaSet":            "replicasets",
		"ServiceAccount":        "serviceaccounts",
		"Ingress":               "ingresses",
	}

	specType, ok := typeMapping[resourceType]
	if !ok {
		return false
	}
	return allowedTypes[specType]
}

// hasPreservationLabel checks if a resource has any preservation labels
func (c *Cleaner) hasPreservationLabel(ctx context.Context, finding korpv1alpha1.Finding, preservationLabels []string) bool {
	if len(preservationLabels) == 0 {
		return false
	}

	labels, err := c.getResourceLabels(ctx, finding)
	if err != nil {
		c.logger.Error(err, "Failed to get resource labels, skipping preservation check")
		return false
	}

	for _, preserveLabel := range preservationLabels {
		if _, exists := labels[preserveLabel]; exists {
			return true
		}
	}

	return false
}

// getResourceLabels retrieves labels for a resource
func (c *Cleaner) getResourceLabels(ctx context.Context, finding korpv1alpha1.Finding) (map[string]string, error) {
	switch finding.ResourceType {
	case "ConfigMap":
		obj, err := c.client.CoreV1().ConfigMaps(finding.Namespace).Get(ctx, finding.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return obj.Labels, nil
	case "Secret":
		obj, err := c.client.CoreV1().Secrets(finding.Namespace).Get(ctx, finding.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return obj.Labels, nil
	case "PersistentVolumeClaim":
		obj, err := c.client.CoreV1().PersistentVolumeClaims(finding.Namespace).Get(ctx, finding.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return obj.Labels, nil
	case "Service":
		obj, err := c.client.CoreV1().Services(finding.Namespace).Get(ctx, finding.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return obj.Labels, nil
	case "Deployment":
		obj, err := c.client.AppsV1().Deployments(finding.Namespace).Get(ctx, finding.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return obj.Labels, nil
	case "StatefulSet":
		obj, err := c.client.AppsV1().StatefulSets(finding.Namespace).Get(ctx, finding.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return obj.Labels, nil
	case "DaemonSet":
		obj, err := c.client.AppsV1().DaemonSets(finding.Namespace).Get(ctx, finding.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return obj.Labels, nil
	case "Job":
		obj, err := c.client.BatchV1().Jobs(finding.Namespace).Get(ctx, finding.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return obj.Labels, nil
	case "CronJob":
		obj, err := c.client.BatchV1().CronJobs(finding.Namespace).Get(ctx, finding.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return obj.Labels, nil
	case "ReplicaSet":
		obj, err := c.client.AppsV1().ReplicaSets(finding.Namespace).Get(ctx, finding.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return obj.Labels, nil
	case "ServiceAccount":
		obj, err := c.client.CoreV1().ServiceAccounts(finding.Namespace).Get(ctx, finding.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return obj.Labels, nil
	case "Ingress":
		obj, err := c.client.NetworkingV1().Ingresses(finding.Namespace).Get(ctx, finding.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return obj.Labels, nil
	default:
		return nil, fmt.Errorf("unsupported resource type: %s", finding.ResourceType)
	}
}

// deleteResource deletes a resource based on its type
func (c *Cleaner) deleteResource(ctx context.Context, finding korpv1alpha1.Finding) error {
	deletePolicy := metav1.DeletePropagationBackground

	switch finding.ResourceType {
	case "ConfigMap":
		return c.client.CoreV1().ConfigMaps(finding.Namespace).Delete(ctx, finding.Name, metav1.DeleteOptions{PropagationPolicy: &deletePolicy})
	case "Secret":
		return c.client.CoreV1().Secrets(finding.Namespace).Delete(ctx, finding.Name, metav1.DeleteOptions{PropagationPolicy: &deletePolicy})
	case "PersistentVolumeClaim":
		return c.client.CoreV1().PersistentVolumeClaims(finding.Namespace).Delete(ctx, finding.Name, metav1.DeleteOptions{PropagationPolicy: &deletePolicy})
	case "Service":
		return c.client.CoreV1().Services(finding.Namespace).Delete(ctx, finding.Name, metav1.DeleteOptions{PropagationPolicy: &deletePolicy})
	case "Deployment":
		return c.client.AppsV1().Deployments(finding.Namespace).Delete(ctx, finding.Name, metav1.DeleteOptions{PropagationPolicy: &deletePolicy})
	case "StatefulSet":
		return c.client.AppsV1().StatefulSets(finding.Namespace).Delete(ctx, finding.Name, metav1.DeleteOptions{PropagationPolicy: &deletePolicy})
	case "DaemonSet":
		return c.client.AppsV1().DaemonSets(finding.Namespace).Delete(ctx, finding.Name, metav1.DeleteOptions{PropagationPolicy: &deletePolicy})
	case "Job":
		return c.client.BatchV1().Jobs(finding.Namespace).Delete(ctx, finding.Name, metav1.DeleteOptions{PropagationPolicy: &deletePolicy})
	case "CronJob":
		return c.client.BatchV1().CronJobs(finding.Namespace).Delete(ctx, finding.Name, metav1.DeleteOptions{PropagationPolicy: &deletePolicy})
	case "ReplicaSet":
		return c.client.AppsV1().ReplicaSets(finding.Namespace).Delete(ctx, finding.Name, metav1.DeleteOptions{PropagationPolicy: &deletePolicy})
	case "ServiceAccount":
		return c.client.CoreV1().ServiceAccounts(finding.Namespace).Delete(ctx, finding.Name, metav1.DeleteOptions{PropagationPolicy: &deletePolicy})
	case "Ingress":
		return c.client.NetworkingV1().Ingresses(finding.Namespace).Delete(ctx, finding.Name, metav1.DeleteOptions{PropagationPolicy: &deletePolicy})
	default:
		return fmt.Errorf("unsupported resource type for deletion: %s", finding.ResourceType)
	}
}
