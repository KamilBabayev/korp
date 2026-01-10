package k8s

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// OrphanConfigMaps returns names of ConfigMaps without ownerReferences and not used by any pods.
func OrphanConfigMaps(ctx context.Context, client *kubernetes.Clientset, ns string) ([]string, error) {
	cms, err := client.CoreV1().ConfigMaps(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	// Get all pods in the namespace
	pods, err := client.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var names []string
	for _, cm := range cms.Items {
		// Skip if it has owner references
		if len(cm.OwnerReferences) > 0 {
			continue
		}

		// Check if any pod is using this ConfigMap
		isUsed := false
		for _, pod := range pods.Items {
			if isConfigMapUsedByPod(pod, cm.Name) {
				isUsed = true
				break
			}
		}

		// Only report as orphan if not used by any pod
		if !isUsed {
			names = append(names, cm.Name)
		}
	}
	return names, nil
}

// OrphanSecrets returns names of Secrets without ownerReferences and not used by any pods.
func OrphanSecrets(ctx context.Context, client *kubernetes.Clientset, ns string) ([]string, error) {
	items, err := client.CoreV1().Secrets(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	// Get all pods in the namespace
	pods, err := client.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var names []string
	for _, s := range items.Items {
		// Skip if it has owner references
		if len(s.OwnerReferences) > 0 {
			continue
		}

		// Check if any pod is using this Secret
		isUsed := false
		for _, pod := range pods.Items {
			if isSecretUsedByPod(pod, s.Name) {
				isUsed = true
				break
			}
		}

		// Only report as orphan if not used by any pod
		if !isUsed {
			names = append(names, s.Name)
		}
	}
	return names, nil
}

// OrphanPVCs returns names of PersistentVolumeClaims without ownerReferences and not used by any pods.
func OrphanPVCs(ctx context.Context, client *kubernetes.Clientset, ns string) ([]string, error) {
	items, err := client.CoreV1().PersistentVolumeClaims(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	// Get all pods in the namespace
	pods, err := client.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var names []string
	for _, p := range items.Items {
		// Skip if it has owner references
		if len(p.OwnerReferences) > 0 {
			continue
		}

		// Check if any pod is using this PVC
		isUsed := false
		for _, pod := range pods.Items {
			if isPVCUsedByPod(pod, p.Name) {
				isUsed = true
				break
			}
		}

		// Only report as orphan if not used by any pod
		if !isUsed {
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

// isConfigMapUsedByPod checks if a ConfigMap is referenced by a pod
func isConfigMapUsedByPod(pod corev1.Pod, configMapName string) bool {
	// Check volumes
	for _, vol := range pod.Spec.Volumes {
		if vol.ConfigMap != nil && vol.ConfigMap.Name == configMapName {
			return true
		}
		if vol.Projected != nil {
			for _, source := range vol.Projected.Sources {
				if source.ConfigMap != nil && source.ConfigMap.Name == configMapName {
					return true
				}
			}
		}
	}

	// Check all containers (including init and ephemeral)
	allContainers := append([]corev1.Container{}, pod.Spec.InitContainers...)
	allContainers = append(allContainers, pod.Spec.Containers...)
	for _, ec := range pod.Spec.EphemeralContainers {
		allContainers = append(allContainers, corev1.Container{
			Env:     ec.Env,
			EnvFrom: ec.EnvFrom,
		})
	}

	for _, container := range allContainers {
		// Check envFrom
		for _, envFrom := range container.EnvFrom {
			if envFrom.ConfigMapRef != nil && envFrom.ConfigMapRef.Name == configMapName {
				return true
			}
		}
		// Check env
		for _, env := range container.Env {
			if env.ValueFrom != nil && env.ValueFrom.ConfigMapKeyRef != nil {
				if env.ValueFrom.ConfigMapKeyRef.Name == configMapName {
					return true
				}
			}
		}
	}

	return false
}

// isSecretUsedByPod checks if a Secret is referenced by a pod
func isSecretUsedByPod(pod corev1.Pod, secretName string) bool {
	// Check volumes
	for _, vol := range pod.Spec.Volumes {
		if vol.Secret != nil && vol.Secret.SecretName == secretName {
			return true
		}
		if vol.Projected != nil {
			for _, source := range vol.Projected.Sources {
				if source.Secret != nil && source.Secret.Name == secretName {
					return true
				}
			}
		}
	}

	// Check imagePullSecrets
	for _, ips := range pod.Spec.ImagePullSecrets {
		if ips.Name == secretName {
			return true
		}
	}

	// Check all containers (including init and ephemeral)
	allContainers := append([]corev1.Container{}, pod.Spec.InitContainers...)
	allContainers = append(allContainers, pod.Spec.Containers...)
	for _, ec := range pod.Spec.EphemeralContainers {
		allContainers = append(allContainers, corev1.Container{
			Env:     ec.Env,
			EnvFrom: ec.EnvFrom,
		})
	}

	for _, container := range allContainers {
		// Check envFrom
		for _, envFrom := range container.EnvFrom {
			if envFrom.SecretRef != nil && envFrom.SecretRef.Name == secretName {
				return true
			}
		}
		// Check env
		for _, env := range container.Env {
			if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil {
				if env.ValueFrom.SecretKeyRef.Name == secretName {
					return true
				}
			}
		}
	}

	return false
}

// isPVCUsedByPod checks if a PVC is referenced by a pod
func isPVCUsedByPod(pod corev1.Pod, pvcName string) bool {
	for _, vol := range pod.Spec.Volumes {
		if vol.PersistentVolumeClaim != nil && vol.PersistentVolumeClaim.ClaimName == pvcName {
			return true
		}
	}
	return false
}
