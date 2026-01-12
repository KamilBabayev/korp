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
}

// FilterSpec defines filtering rules for excluding resources
type FilterSpec struct {
	// ExcludeLabels are label selectors to exclude
	// +optional
	ExcludeLabels map[string]string `json:"excludeLabels,omitempty"`

	// ExcludeNamePatterns are regex patterns to exclude by name
	// +optional
	ExcludeNamePatterns []string `json:"excludeNamePatterns,omitempty"`
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

// ScanSummary provides aggregate counts of orphaned resources
type ScanSummary struct {
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
}

// TotalOrphans returns the sum of all orphaned resources
func (s *ScanSummary) TotalOrphans() int {
	return s.OrphanedConfigMaps + s.OrphanedSecrets + s.OrphanedPVCs +
		s.ServicesWithoutEndpoints + s.OrphanedDeployments +
		s.OrphanedJobs + s.OrphanedIngresses
}

// Finding represents a single orphaned resource
type Finding struct {
	// ResourceType is the type of resource (ConfigMap, Secret, etc.)
	ResourceType string `json:"resourceType"`

	// Namespace is the namespace of the resource
	Namespace string `json:"namespace"`

	// Name is the name of the resource
	Name string `json:"name"`

	// Reason explains why the resource is considered orphaned
	Reason string `json:"reason"`

	// DetectedAt is when this orphan was detected
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
// +kubebuilder:printcolumn:name="Orphans",type=integer,JSONPath=`.status.summary.orphanedConfigMaps`
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
