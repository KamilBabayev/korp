/*
Copyright 2026 The Korp Authors.

Licensed under the MIT License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KorpScanSpec defines the desired state of KorpScan
type KorpScanSpec struct {
	// TargetNamespace is the namespace to scan. Use "*" for all namespaces.
	// +kubebuilder:validation:Required
	TargetNamespace string `json:"targetNamespace"`

	// IntervalMinutes is the scan interval in minutes
	// +kubebuilder:default=60
	// +kubebuilder:validation:Minimum=1
	// +optional
	IntervalMinutes int `json:"intervalMinutes,omitempty"`

	// ResourceTypes to scan. Defaults to all if empty.
	// +kubebuilder:validation:Optional
	// +optional
	ResourceTypes []string `json:"resourceTypes,omitempty"`

	// Filters for excluding resources
	// +kubebuilder:validation:Optional
	// +optional
	Filters FilterSpec `json:"filters,omitempty"`

	// Reporting configuration
	// +kubebuilder:validation:Optional
	// +optional
	Reporting ReportingSpec `json:"reporting,omitempty"`

	// Cleanup configuration for automatic resource cleanup
	// +kubebuilder:validation:Optional
	// +optional
	Cleanup *CleanupSpec `json:"cleanup,omitempty"`
}

// FilterSpec defines filtering rules for excluding resources
type FilterSpec struct {
	// ExcludeLabels are label selectors to exclude
	// +optional
	ExcludeLabels map[string]string `json:"excludeLabels,omitempty"`

	// ExcludeNamePatterns are regex patterns to exclude by name
	// +optional
	ExcludeNamePatterns []string `json:"excludeNamePatterns,omitempty"`

	// ExcludeNamespaces are namespaces to completely exclude from scanning
	// +optional
	ExcludeNamespaces []string `json:"excludeNamespaces,omitempty"`
}

// ReportingSpec defines how scan results are reported
type ReportingSpec struct {
	// CreateEvents determines if Kubernetes events should be created
	// +kubebuilder:default=true
	// +optional
	CreateEvents bool `json:"createEvents,omitempty"`

	// EventSeverity is the event severity (Normal or Warning)
	// +kubebuilder:default="Warning"
	// +kubebuilder:validation:Enum=Normal;Warning
	// +optional
	EventSeverity string `json:"eventSeverity,omitempty"`

	// HistoryLimit is the number of scan results to retain
	// +kubebuilder:default=5
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=50
	// +optional
	HistoryLimit int `json:"historyLimit,omitempty"`

	// Webhook configuration for sending scan results to external systems
	// +optional
	Webhook *WebhookConfig `json:"webhook,omitempty"`
}

// WebhookConfig defines webhook notification settings
type WebhookConfig struct {
	// URL is the webhook endpoint to send notifications to
	// +kubebuilder:validation:Required
	URL string `json:"url"`

	// Method is the HTTP method to use (default: POST)
	// +kubebuilder:default="POST"
	// +kubebuilder:validation:Enum=POST;PUT
	// +optional
	Method string `json:"method,omitempty"`

	// Headers are custom HTTP headers to include in the webhook request
	// +optional
	Headers map[string]string `json:"headers,omitempty"`

	// TimeoutSeconds is the request timeout in seconds (default: 30)
	// +kubebuilder:default=30
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=300
	// +optional
	TimeoutSeconds int `json:"timeoutSeconds,omitempty"`

	// InsecureSkipVerify skips TLS certificate verification (not recommended)
	// +kubebuilder:default=false
	// +optional
	InsecureSkipVerify bool `json:"insecureSkipVerify,omitempty"`

	// RetryPolicy defines retry behavior for failed webhook calls
	// +optional
	RetryPolicy *RetryPolicy `json:"retryPolicy,omitempty"`
}

// RetryPolicy defines retry behavior for webhook notifications
type RetryPolicy struct {
	// MaxRetries is the maximum number of retry attempts (default: 3)
	// +kubebuilder:default=3
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=10
	// +optional
	MaxRetries int `json:"maxRetries,omitempty"`

	// InitialDelaySeconds is the initial delay before first retry in seconds (default: 1)
	// +kubebuilder:default=1
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=60
	// +optional
	InitialDelaySeconds int `json:"initialDelaySeconds,omitempty"`
}

// CleanupSpec defines automatic cleanup configuration
type CleanupSpec struct {
	// Enabled determines if automatic cleanup is enabled
	// +kubebuilder:default=false
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// DryRun when true, only logs what would be deleted without actually deleting
	// IMPORTANT: Default is true for safety - must explicitly set to false to delete
	// +kubebuilder:default=true
	// +optional
	DryRun *bool `json:"dryRun,omitempty"`

	// ResourceTypes specifies which resource types to clean up
	// If empty, all detected orphan types are eligible for cleanup
	// +optional
	ResourceTypes []string `json:"resourceTypes,omitempty"`

	// MinAgeDays is the minimum age in days before a resource is eligible for cleanup
	// Resources must be orphaned for at least this many days before deletion
	// +kubebuilder:default=7
	// +kubebuilder:validation:Minimum=0
	// +optional
	MinAgeDays int `json:"minAgeDays,omitempty"`

	// PreservationLabels are label keys that, when present on a resource, prevent cleanup
	// Example: "korp.io/preserve", "do-not-delete"
	// +optional
	PreservationLabels []string `json:"preservationLabels,omitempty"`
}

// IsDryRun returns true if dry-run mode is enabled (default: true for safety)
func (c *CleanupSpec) IsDryRun() bool {
	if c.DryRun == nil {
		return true
	}
	return *c.DryRun
}

// KorpScanStatus defines the observed state of KorpScan
type KorpScanStatus struct {
	// LastScanTime is when the last scan completed
	// +optional
	LastScanTime *metav1.Time `json:"lastScanTime,omitempty"`

	// Phase represents the current state
	// +kubebuilder:validation:Enum=Pending;Running;Completed;Failed
	// +optional
	Phase string `json:"phase,omitempty"`

	// Summary of findings
	// +optional
	Summary ScanSummary `json:"summary,omitempty"`

	// Findings contains detailed orphan resource information
	// +optional
	Findings []Finding `json:"findings,omitempty"`

	// History of recent scans
	// +optional
	History []HistoryEntry `json:"history,omitempty"`

	// Conditions represent the latest observations
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// WebhookStatus tracks webhook notification status
	// +optional
	WebhookStatus *WebhookStatus `json:"webhookStatus,omitempty"`

	// CleanupStatus tracks cleanup operation status
	// +optional
	CleanupStatus *CleanupStatus `json:"cleanupStatus,omitempty"`
}

// WebhookStatus tracks the status of webhook notifications
type WebhookStatus struct {
	// LastSuccess is the timestamp of the last successful webhook delivery
	// +optional
	LastSuccess *metav1.Time `json:"lastSuccess,omitempty"`

	// LastFailure is the timestamp of the last failed webhook delivery
	// +optional
	LastFailure *metav1.Time `json:"lastFailure,omitempty"`

	// FailureCount is the number of consecutive webhook failures
	// +optional
	FailureCount int `json:"failureCount,omitempty"`

	// LastError contains the error message from the last failed webhook
	// +optional
	LastError string `json:"lastError,omitempty"`
}

// CleanupStatus tracks the status of cleanup operations
type CleanupStatus struct {
	// LastCleanupTime is when the last cleanup operation completed
	// +optional
	LastCleanupTime *metav1.Time `json:"lastCleanupTime,omitempty"`

	// LastCleanupResult indicates the result of the last cleanup (Success, Failed, DryRun)
	// +optional
	LastCleanupResult string `json:"lastCleanupResult,omitempty"`

	// Summary of the last cleanup operation
	// +optional
	Summary *CleanupSummary `json:"summary,omitempty"`

	// DeletedResources lists resources that were deleted in the last cleanup
	// +optional
	DeletedResources []DeletedResource `json:"deletedResources,omitempty"`

	// FailedDeletions lists resources that failed to delete
	// +optional
	FailedDeletions []FailedDeletion `json:"failedDeletions,omitempty"`
}

// CleanupSummary provides aggregate counts for cleanup operations
type CleanupSummary struct {
	// TotalEligible is the number of resources eligible for cleanup
	TotalEligible int `json:"totalEligible"`

	// TotalDeleted is the number of resources actually deleted
	TotalDeleted int `json:"totalDeleted"`

	// TotalFailed is the number of failed deletion attempts
	TotalFailed int `json:"totalFailed"`

	// TotalSkippedPreserved is the count skipped due to preservation labels
	TotalSkippedPreserved int `json:"totalSkippedPreserved"`

	// TotalSkippedAge is the count skipped due to age threshold
	TotalSkippedAge int `json:"totalSkippedAge"`

	// DryRun indicates if this was a dry-run operation
	DryRun bool `json:"dryRun"`
}

// DeletedResource represents a resource that was deleted
type DeletedResource struct {
	// ResourceType is the type of resource (ConfigMap, Secret, etc.)
	ResourceType string `json:"resourceType"`

	// Namespace is the namespace of the deleted resource
	Namespace string `json:"namespace"`

	// Name is the name of the deleted resource
	Name string `json:"name"`

	// DeletedAt is when the resource was deleted
	DeletedAt metav1.Time `json:"deletedAt"`
}

// FailedDeletion represents a resource that failed to delete
type FailedDeletion struct {
	// ResourceType is the type of resource
	ResourceType string `json:"resourceType"`

	// Namespace is the namespace of the resource
	Namespace string `json:"namespace"`

	// Name is the name of the resource
	Name string `json:"name"`

	// Error is the error message explaining the failure
	Error string `json:"error"`
}

// ScanSummary provides aggregate counts of orphaned resources
type ScanSummary struct {
	// OrphanCount is the total number of orphaned resources found
	// +optional
	OrphanCount int `json:"orphanCount,omitempty"`

	// TotalResources is the total number of resources scanned
	TotalResources int `json:"totalResources"`

	// OrphanedConfigMaps is the count of orphaned ConfigMaps
	OrphanedConfigMaps int `json:"orphanedConfigMaps"`

	// OrphanedSecrets is the count of orphaned Secrets
	OrphanedSecrets int `json:"orphanedSecrets"`

	// OrphanedPVCs is the count of orphaned PersistentVolumeClaims
	OrphanedPVCs int `json:"orphanedPVCs"`

	// ServicesWithoutEndpoints is the count of Services without Endpoints
	ServicesWithoutEndpoints int `json:"servicesWithoutEndpoints"`

	// OrphanedDeployments is the count of orphaned Deployments
	// +optional
	OrphanedDeployments int `json:"orphanedDeployments,omitempty"`

	// OrphanedJobs is the count of orphaned Jobs
	// +optional
	OrphanedJobs int `json:"orphanedJobs,omitempty"`

	// OrphanedIngresses is the count of orphaned Ingresses
	// +optional
	OrphanedIngresses int `json:"orphanedIngresses,omitempty"`

	// OrphanedStatefulSets is the count of orphaned StatefulSets
	// +optional
	OrphanedStatefulSets int `json:"orphanedStatefulSets,omitempty"`

	// OrphanedDaemonSets is the count of orphaned DaemonSets
	// +optional
	OrphanedDaemonSets int `json:"orphanedDaemonSets,omitempty"`

	// OrphanedCronJobs is the count of orphaned CronJobs
	// +optional
	OrphanedCronJobs int `json:"orphanedCronJobs,omitempty"`

	// OrphanedReplicaSets is the count of orphaned ReplicaSets
	// +optional
	OrphanedReplicaSets int `json:"orphanedReplicaSets,omitempty"`

	// OrphanedServiceAccounts is the count of orphaned ServiceAccounts
	// +optional
	OrphanedServiceAccounts int `json:"orphanedServiceAccounts,omitempty"`

	// OrphanedRoles is the count of orphaned Roles (not referenced by any RoleBinding)
	// +optional
	OrphanedRoles int `json:"orphanedRoles,omitempty"`

	// OrphanedClusterRoles is the count of orphaned ClusterRoles (not referenced by any binding)
	// +optional
	OrphanedClusterRoles int `json:"orphanedClusterRoles,omitempty"`

	// OrphanedRoleBindings is the count of orphaned RoleBindings (referencing non-existent roles/subjects)
	// +optional
	OrphanedRoleBindings int `json:"orphanedRoleBindings,omitempty"`

	// OrphanedClusterRoleBindings is the count of orphaned ClusterRoleBindings
	// +optional
	OrphanedClusterRoleBindings int `json:"orphanedClusterRoleBindings,omitempty"`

	// OrphanedNetworkPolicies is the count of orphaned NetworkPolicies (selector matches no pods)
	// +optional
	OrphanedNetworkPolicies int `json:"orphanedNetworkPolicies,omitempty"`

	// OrphanedPodDisruptionBudgets is the count of orphaned PodDisruptionBudgets (selector matches no pods)
	// +optional
	OrphanedPodDisruptionBudgets int `json:"orphanedPodDisruptionBudgets,omitempty"`

	// OrphanedHPAs is the count of orphaned HorizontalPodAutoscalers (targeting non-existent workloads)
	// +optional
	OrphanedHPAs int `json:"orphanedHPAs,omitempty"`

	// OrphanedPVs is the count of orphaned PersistentVolumes (Released or Available state)
	// +optional
	OrphanedPVs int `json:"orphanedPVs,omitempty"`

	// OrphanedEndpoints is the count of orphaned Endpoints (no corresponding Service)
	// +optional
	OrphanedEndpoints int `json:"orphanedEndpoints,omitempty"`

	// OrphanedResourceQuotas is the count of orphaned ResourceQuotas (namespace has no pods)
	// +optional
	OrphanedResourceQuotas int `json:"orphanedResourceQuotas,omitempty"`
}

// TotalOrphans returns the sum of all orphaned resources
func (s *ScanSummary) TotalOrphans() int {
	return s.OrphanedConfigMaps + s.OrphanedSecrets + s.OrphanedPVCs +
		s.ServicesWithoutEndpoints + s.OrphanedDeployments +
		s.OrphanedJobs + s.OrphanedIngresses +
		s.OrphanedStatefulSets + s.OrphanedDaemonSets +
		s.OrphanedCronJobs + s.OrphanedReplicaSets +
		s.OrphanedServiceAccounts + s.OrphanedRoles +
		s.OrphanedClusterRoles + s.OrphanedRoleBindings +
		s.OrphanedClusterRoleBindings + s.OrphanedNetworkPolicies +
		s.OrphanedPodDisruptionBudgets + s.OrphanedHPAs +
		s.OrphanedPVs + s.OrphanedEndpoints + s.OrphanedResourceQuotas
}

// Finding represents a single orphaned resource
type Finding struct {
	// Separator is a visual divider between findings
	// +optional
	Separator string `json:"---,omitempty"`

	// Description is a one-line summary: "ConfigMap korp/name (Reason)"
	// +optional
	Description string `json:"description,omitempty"`

	// ResourceType is the kind of resource (ConfigMap, Secret, Service, etc.)
	ResourceType string `json:"resourceType"`

	// Name is the name of the orphaned resource
	Name string `json:"name"`

	// Namespace where the resource is located
	Namespace string `json:"namespace"`

	// Reason explains why this resource is considered orphaned
	Reason string `json:"reason"`

	// DetectedAt timestamp when this orphan was first detected
	DetectedAt metav1.Time `json:"detectedAt"`
}

// HistoryEntry represents a historical scan result
type HistoryEntry struct {
	// ScanTime is when the scan completed
	ScanTime metav1.Time `json:"scanTime"`

	// OrphanCount is the number of orphans found
	OrphanCount int `json:"orphanCount"`

	// Duration is how long the scan took
	Duration string `json:"duration"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:printcolumn:name="Target",type=string,JSONPath=`.spec.targetNamespace`
// +kubebuilder:printcolumn:name="Interval",type=integer,JSONPath=`.spec.intervalMinutes`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Orphans",type=integer,JSONPath=`.status.summary.orphanCount`
// +kubebuilder:printcolumn:name="Services",type=integer,JSONPath=`.status.summary.servicesWithoutEndpoints`,priority=1
// +kubebuilder:printcolumn:name="Deploys",type=integer,JSONPath=`.status.summary.orphanedDeployments`,priority=1
// +kubebuilder:printcolumn:name="LastScan",type=date,JSONPath=`.status.lastScanTime`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// KorpScan is the Schema for the korpscans API
type KorpScan struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KorpScanSpec   `json:"spec,omitempty"`
	Status KorpScanStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// KorpScanList contains a list of KorpScan
type KorpScanList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KorpScan `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KorpScan{}, &KorpScanList{})
}
