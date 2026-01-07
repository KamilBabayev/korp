/*
Copyright 2026 The Korp Authors.

Licensed under the MIT License.
*/

package controller

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	korpv1alpha1 "github.com/kamilbabayev/korp/api/v1alpha1"
	"github.com/kamilbabayev/korp/pkg/reporter"
	"github.com/kamilbabayev/korp/pkg/scan"
)

// KorpScanReconciler reconciles a KorpScan object
type KorpScanReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	Clientset *kubernetes.Clientset
	Scanner   *scan.Scanner
	Reporter  *reporter.EventReporter
}

// +kubebuilder:rbac:groups=korp.io,resources=korpscans,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=korp.io,resources=korpscans/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=korp.io,resources=korpscans/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list
// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list
// +kubebuilder:rbac:groups="",resources=endpoints,verbs=get;list
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// Reconcile is the main reconciliation loop
func (r *KorpScanReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the KorpScan resource
	var korpScan korpv1alpha1.KorpScan
	if err := r.Get(ctx, req.NamespacedName, &korpScan); err != nil {
		if errors.IsNotFound(err) {
			// Resource was deleted, nothing to do
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get KorpScan")
		return ctrl.Result{}, err
	}

	// Determine scan interval
	interval := time.Duration(korpScan.Spec.IntervalMinutes) * time.Minute
	if interval == 0 {
		interval = 60 * time.Minute // Default to 60 minutes
	}

	// Check if scan is due
	if korpScan.Status.LastScanTime != nil {
		nextScan := korpScan.Status.LastScanTime.Add(interval)
		if time.Now().Before(nextScan) {
			requeueAfter := time.Until(nextScan)
			log.Info("Scan not due yet", "requeueAfter", requeueAfter)
			return ctrl.Result{RequeueAfter: requeueAfter}, nil
		}
	}

	// Update status to Running
	korpScan.Status.Phase = "Running"
	if err := r.Status().Update(ctx, &korpScan); err != nil {
		log.Error(err, "Failed to update status to Running")
		return ctrl.Result{}, err
	}

	// Perform scan
	log.Info("Starting scan", "targetNamespace", korpScan.Spec.TargetNamespace)
	startTime := time.Now()

	result, err := r.Scanner.Scan(ctx, &korpScan)
	if err != nil {
		log.Error(err, "Scan failed")
		korpScan.Status.Phase = "Failed"
		r.updateCondition(&korpScan, "Ready", metav1.ConditionFalse, "ScanFailed", err.Error())
		if statusErr := r.Status().Update(ctx, &korpScan); statusErr != nil {
			log.Error(statusErr, "Failed to update status after scan failure")
		}
		return ctrl.Result{RequeueAfter: interval}, err
	}

	duration := time.Since(startTime)
	log.Info("Scan completed", "duration", duration, "orphans", len(result.Details))

	// Update status with results
	now := metav1.Time{Time: time.Now()}
	korpScan.Status.LastScanTime = &now
	korpScan.Status.Phase = "Completed"
	korpScan.Status.Summary = result.Summary
	korpScan.Status.Findings = result.Details

	// Add to history
	historyLimit := korpScan.Spec.Reporting.HistoryLimit
	if historyLimit == 0 {
		historyLimit = 5
	}

	totalOrphans := result.Summary.TotalOrphans()
	korpScan.Status.History = append([]korpv1alpha1.HistoryEntry{{
		ScanTime:    now,
		OrphanCount: totalOrphans,
		Duration:    duration.String(),
	}}, korpScan.Status.History...)

	if len(korpScan.Status.History) > historyLimit {
		korpScan.Status.History = korpScan.Status.History[:historyLimit]
	}

	// Update condition
	r.updateCondition(&korpScan, "Ready", metav1.ConditionTrue, "ScanCompleted",
		fmt.Sprintf("Found %d orphaned resources", totalOrphans))

	// Update status
	if err := r.Status().Update(ctx, &korpScan); err != nil {
		log.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	// Create events if enabled
	if korpScan.Spec.Reporting.CreateEvents {
		r.Reporter.CreateEvents(ctx, &korpScan, result)
	}

	// Requeue for next scan
	log.Info("Scan completed successfully", "nextScanIn", interval)
	return ctrl.Result{RequeueAfter: interval}, nil
}

// updateCondition updates or adds a condition to the KorpScan status
func (r *KorpScanReconciler) updateCondition(korpScan *korpv1alpha1.KorpScan,
	condType string, status metav1.ConditionStatus, reason, message string) {

	meta.SetStatusCondition(&korpScan.Status.Conditions, metav1.Condition{
		Type:               condType,
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: korpScan.Generation,
		LastTransitionTime: metav1.Now(),
	})
}

// SetupWithManager sets up the controller with the Manager
func (r *KorpScanReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&korpv1alpha1.KorpScan{}).
		Complete(r)
}
