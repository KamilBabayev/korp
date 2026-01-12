/*
Copyright 2026 The Korp Authors.

Licensed under the MIT License.
*/

package notifier

import (
	"github.com/kamilbabayev/korp/api/v1alpha1"
)

// WebhookPayload represents the JSON payload sent to webhook endpoints
type WebhookPayload struct {
	// EventType describes the type of event (e.g., "scan.completed")
	EventType string `json:"eventType"`

	// Timestamp is the ISO8601 formatted time when the event occurred
	Timestamp string `json:"timestamp"`

	// KorpScan contains metadata about the KorpScan resource
	KorpScan ScanMetadata `json:"korpscan"`

	// Summary contains aggregate counts of orphaned resources
	Summary v1alpha1.ScanSummary `json:"summary"`

	// Findings contains detailed information about each orphaned resource
	Findings []v1alpha1.Finding `json:"findings"`

	// ScanDuration is the human-readable duration of the scan (e.g., "2.5s")
	ScanDuration string `json:"scanDuration"`
}

// ScanMetadata contains identifying information about a KorpScan resource
type ScanMetadata struct {
	// Name is the name of the KorpScan resource
	Name string `json:"name"`

	// Namespace is the namespace where the KorpScan resource resides
	Namespace string `json:"namespace"`

	// TargetNamespace is the namespace being scanned
	TargetNamespace string `json:"targetNamespace"`
}
