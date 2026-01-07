package k8s

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// OrphanConfigMaps returns names of ConfigMaps without ownerReferences in the given namespace.
func OrphanConfigMaps(ctx context.Context, client *kubernetes.Clientset, ns string) ([]string, error) {
	cms, err := client.CoreV1().ConfigMaps(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var names []string
	for _, cm := range cms.Items {
		if len(cm.OwnerReferences) == 0 {
			names = append(names, cm.Name)
		}
	}
	return names, nil
}

// OrphanSecrets returns names of Secrets without ownerReferences in the given namespace.
func OrphanSecrets(ctx context.Context, client *kubernetes.Clientset, ns string) ([]string, error) {
	items, err := client.CoreV1().Secrets(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var names []string
	for _, s := range items.Items {
		if len(s.OwnerReferences) == 0 {
			names = append(names, s.Name)
		}
	}
	return names, nil
}

// OrphanPVCs returns names of PersistentVolumeClaims without ownerReferences in the given namespace.
func OrphanPVCs(ctx context.Context, client *kubernetes.Clientset, ns string) ([]string, error) {
	items, err := client.CoreV1().PersistentVolumeClaims(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var names []string
	for _, p := range items.Items {
		if len(p.OwnerReferences) == 0 {
			names = append(names, p.Name)
		}
	}
	return names, nil
}

// ServicesWithoutEndpoints returns service names that currently have no endpoints.
func ServicesWithoutEndpoints(ctx context.Context, client *kubernetes.Clientset, ns string) ([]string, error) {
	svcs, err := client.CoreV1().Services(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var names []string
	for _, svc := range svcs.Items {
		ep, err := client.CoreV1().Endpoints(svc.Namespace).Get(ctx, svc.Name, metav1.GetOptions{})
		if err != nil {
			// missing endpoints resource — treat as no endpoints
			names = append(names, svc.Name)
			continue
		}
		total := 0
		for _, subset := range ep.Subsets {
			total += len(subset.Addresses)
			total += len(subset.NotReadyAddresses)
		}
		if total == 0 {
			names = append(names, svc.Name)
		}
	}
	return names, nil
}
