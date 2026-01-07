/*
Copyright 2026 The Korp Authors.

Licensed under the MIT License.
*/

package scan

import (
	korpv1alpha1 "github.com/kamilbabayev/korp/api/v1alpha1"
)

// ScanResult holds the results of a scan operation
type ScanResult struct {
	// Summary provides aggregate counts
	Summary korpv1alpha1.ScanSummary

	// Details contains individual findings
	Details []korpv1alpha1.Finding
}
