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

// OrphanDeployments returns names of Deployments with 0 replicas or no running pods
func OrphanDeployments(ctx context.Context, client *kubernetes.Clientset, ns string) ([]string, error) {
	deployments, err := client.AppsV1().Deployments(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var names []string
	for _, dep := range deployments.Items {
		// Check if deployment has 0 replicas
		if dep.Spec.Replicas != nil && *dep.Spec.Replicas == 0 {
			names = append(names, dep.Name)
			continue
		}

		// Check if deployment has no ready replicas
		if dep.Status.ReadyReplicas == 0 && dep.Status.Replicas == 0 {
			names = append(names, dep.Name)
		}
	}
	return names, nil
}

// OrphanJobs returns names of completed Jobs older than 7 days
func OrphanJobs(ctx context.Context, client *kubernetes.Clientset, ns string) ([]string, error) {
	jobs, err := client.BatchV1().Jobs(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var names []string
	for _, job := range jobs.Items {
		// Skip if it has owner references (managed by CronJob, etc)
		if len(job.OwnerReferences) > 0 {
			continue
		}

		// Check if job is completed and older than 7 days
		if job.Status.Succeeded > 0 || job.Status.Failed > 0 {
			if job.Status.CompletionTime != nil {
				age := metav1.Now().Sub(job.Status.CompletionTime.Time)
				if age.Hours() > 168 { // 7 days
					names = append(names, job.Name)
				}
			}
		}
	}
	return names, nil
}

// OrphanIngresses returns names of Ingresses pointing to non-existent services
func OrphanIngresses(ctx context.Context, client *kubernetes.Clientset, ns string) ([]string, error) {
	ingresses, err := client.NetworkingV1().Ingresses(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	// Get all services in namespace for quick lookup
	services, err := client.CoreV1().Services(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	serviceMap := make(map[string]bool)
	for _, svc := range services.Items {
		serviceMap[svc.Name] = true
	}

	var names []string
	for _, ing := range ingresses.Items {
		hasValidBackend := false

		// Check default backend
		if ing.Spec.DefaultBackend != nil && ing.Spec.DefaultBackend.Service != nil {
			if serviceMap[ing.Spec.DefaultBackend.Service.Name] {
				hasValidBackend = true
			}
		}

		// Check all rules
		for _, rule := range ing.Spec.Rules {
			if rule.HTTP != nil {
				for _, path := range rule.HTTP.Paths {
					if path.Backend.Service != nil {
						if serviceMap[path.Backend.Service.Name] {
							hasValidBackend = true
							break
						}
					}
				}
			}
			if hasValidBackend {
				break
			}
		}

		// If no valid backend service exists, consider it orphaned
		if !hasValidBackend {
			names = append(names, ing.Name)
		}
	}
	return names, nil
}

// OrphanStatefulSets returns names of StatefulSets with 0 replicas or no ready pods
func OrphanStatefulSets(ctx context.Context, client *kubernetes.Clientset, ns string) ([]string, error) {
	statefulsets, err := client.AppsV1().StatefulSets(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var names []string
	for _, sts := range statefulsets.Items {
		// Check if statefulset has 0 replicas
		if sts.Spec.Replicas != nil && *sts.Spec.Replicas == 0 {
			names = append(names, sts.Name)
			continue
		}

		// Check if statefulset has no ready replicas
		if sts.Status.ReadyReplicas == 0 && sts.Status.Replicas == 0 {
			names = append(names, sts.Name)
		}
	}
	return names, nil
}

// OrphanDaemonSets returns names of DaemonSets with no scheduled pods
func OrphanDaemonSets(ctx context.Context, client *kubernetes.Clientset, ns string) ([]string, error) {
	daemonsets, err := client.AppsV1().DaemonSets(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var names []string
	for _, ds := range daemonsets.Items {
		// Check if daemonset has no scheduled or ready pods
		if ds.Status.DesiredNumberScheduled == 0 || ds.Status.NumberReady == 0 {
			names = append(names, ds.Name)
		}
	}
	return names, nil
}

// OrphanCronJobs returns names of CronJobs that are suspended with no recent successful jobs
func OrphanCronJobs(ctx context.Context, client *kubernetes.Clientset, ns string) ([]string, error) {
	cronjobs, err := client.BatchV1().CronJobs(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var names []string
	for _, cj := range cronjobs.Items {
		// Check if cronjob is suspended
		if cj.Spec.Suspend != nil && *cj.Spec.Suspend {
			// Check if no recent successful job (no last schedule time or very old)
			if cj.Status.LastSuccessfulTime == nil {
				names = append(names, cj.Name)
				continue
			}

			// Consider orphaned if last success was more than 30 days ago
			age := metav1.Now().Sub(cj.Status.LastSuccessfulTime.Time)
			if age.Hours() > 720 { // 30 days
				names = append(names, cj.Name)
			}
		}
	}
	return names, nil
}

// OrphanReplicaSets returns names of ReplicaSets orphaned from deleted Deployments
func OrphanReplicaSets(ctx context.Context, client *kubernetes.Clientset, ns string) ([]string, error) {
	replicasets, err := client.AppsV1().ReplicaSets(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var names []string
	for _, rs := range replicasets.Items {
		// Skip if it has owner references (managed by Deployment)
		if len(rs.OwnerReferences) > 0 {
			continue
		}

		// Orphaned ReplicaSet - no owner and either 0 replicas or no ready pods
		if (rs.Spec.Replicas != nil && *rs.Spec.Replicas == 0) ||
			(rs.Status.ReadyReplicas == 0 && rs.Status.Replicas == 0) {
			names = append(names, rs.Name)
		}
	}
	return names, nil
}

// OrphanServiceAccounts returns names of ServiceAccounts not used by any pod
func OrphanServiceAccounts(ctx context.Context, client *kubernetes.Clientset, ns string) ([]string, error) {
	serviceaccounts, err := client.CoreV1().ServiceAccounts(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	pods, err := client.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	// Build a set of service accounts used by pods
	usedServiceAccounts := make(map[string]bool)
	for _, pod := range pods.Items {
		saName := pod.Spec.ServiceAccountName
		if saName == "" {
			saName = "default"
		}
		usedServiceAccounts[saName] = true
	}

	var names []string
	for _, sa := range serviceaccounts.Items {
		// Skip default service account
		if sa.Name == "default" {
			continue
		}

		// Check if used by any pod
		if !usedServiceAccounts[sa.Name] {
			names = append(names, sa.Name)
		}
	}
	return names, nil
}

// OrphanRoles returns names of Roles not referenced by any RoleBinding
func OrphanRoles(ctx context.Context, client *kubernetes.Clientset, ns string) ([]string, error) {
	roles, err := client.RbacV1().Roles(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	roleBindings, err := client.RbacV1().RoleBindings(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	// Build set of roles referenced by bindings
	referencedRoles := make(map[string]bool)
	for _, rb := range roleBindings.Items {
		if rb.RoleRef.Kind == "Role" {
			referencedRoles[rb.RoleRef.Name] = true
		}
	}

	var names []string
	for _, role := range roles.Items {
		// Skip system roles (prefixed with system:)
		if len(role.Name) > 7 && role.Name[:7] == "system:" {
			continue
		}

		if !referencedRoles[role.Name] {
			names = append(names, role.Name)
		}
	}
	return names, nil
}

// OrphanClusterRoles returns names of ClusterRoles not referenced by any ClusterRoleBinding or RoleBinding
func OrphanClusterRoles(ctx context.Context, client *kubernetes.Clientset) ([]string, error) {
	clusterRoles, err := client.RbacV1().ClusterRoles().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	clusterRoleBindings, err := client.RbacV1().ClusterRoleBindings().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	// Also check RoleBindings that can reference ClusterRoles
	roleBindings, err := client.RbacV1().RoleBindings("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	// Build set of cluster roles referenced by bindings
	referencedClusterRoles := make(map[string]bool)
	for _, crb := range clusterRoleBindings.Items {
		if crb.RoleRef.Kind == "ClusterRole" {
			referencedClusterRoles[crb.RoleRef.Name] = true
		}
	}
	// RoleBindings can also reference ClusterRoles
	for _, rb := range roleBindings.Items {
		if rb.RoleRef.Kind == "ClusterRole" {
			referencedClusterRoles[rb.RoleRef.Name] = true
		}
	}

	var names []string
	for _, cr := range clusterRoles.Items {
		// Skip system cluster roles
		if len(cr.Name) > 7 && cr.Name[:7] == "system:" {
			continue
		}
		// Skip aggregation roles (they aggregate other roles)
		if cr.AggregationRule != nil {
			continue
		}
		// Skip common built-in roles
		if isBuiltInClusterRole(cr.Name) {
			continue
		}

		if !referencedClusterRoles[cr.Name] {
			names = append(names, cr.Name)
		}
	}
	return names, nil
}

// OrphanRoleBindings returns names of RoleBindings that reference non-existent Roles or ServiceAccounts
func OrphanRoleBindings(ctx context.Context, client *kubernetes.Clientset, ns string) ([]string, error) {
	roleBindings, err := client.RbacV1().RoleBindings(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	roles, err := client.RbacV1().Roles(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	clusterRoles, err := client.RbacV1().ClusterRoles().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	// Build set of existing roles
	existingRoles := make(map[string]bool)
	for _, role := range roles.Items {
		existingRoles[role.Name] = true
	}

	existingClusterRoles := make(map[string]bool)
	for _, cr := range clusterRoles.Items {
		existingClusterRoles[cr.Name] = true
	}

	var names []string
	for _, rb := range roleBindings.Items {
		isOrphan := false

		// Check if referenced role exists
		if rb.RoleRef.Kind == "Role" {
			if !existingRoles[rb.RoleRef.Name] {
				isOrphan = true
			}
		} else if rb.RoleRef.Kind == "ClusterRole" {
			if !existingClusterRoles[rb.RoleRef.Name] {
				isOrphan = true
			}
		}

		// Check if any subject references non-existent ServiceAccount in same namespace
		if !isOrphan {
			for _, subject := range rb.Subjects {
				if subject.Kind == "ServiceAccount" {
					subjectNs := subject.Namespace
					if subjectNs == "" {
						subjectNs = ns
					}
					if subjectNs == ns {
						_, err := client.CoreV1().ServiceAccounts(subjectNs).Get(ctx, subject.Name, metav1.GetOptions{})
						if err != nil {
							isOrphan = true
							break
						}
					}
				}
			}
		}

		if isOrphan {
			names = append(names, rb.Name)
		}
	}
	return names, nil
}

// OrphanClusterRoleBindings returns names of ClusterRoleBindings that reference non-existent ClusterRoles or ServiceAccounts
func OrphanClusterRoleBindings(ctx context.Context, client *kubernetes.Clientset) ([]string, error) {
	clusterRoleBindings, err := client.RbacV1().ClusterRoleBindings().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	clusterRoles, err := client.RbacV1().ClusterRoles().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	// Build set of existing cluster roles
	existingClusterRoles := make(map[string]bool)
	for _, cr := range clusterRoles.Items {
		existingClusterRoles[cr.Name] = true
	}

	var names []string
	for _, crb := range clusterRoleBindings.Items {
		// Skip system cluster role bindings
		if len(crb.Name) > 7 && crb.Name[:7] == "system:" {
			continue
		}

		isOrphan := false

		// Check if referenced cluster role exists
		if !existingClusterRoles[crb.RoleRef.Name] {
			isOrphan = true
		}

		// Check if any subject references non-existent ServiceAccount
		if !isOrphan {
			for _, subject := range crb.Subjects {
				if subject.Kind == "ServiceAccount" && subject.Namespace != "" {
					_, err := client.CoreV1().ServiceAccounts(subject.Namespace).Get(ctx, subject.Name, metav1.GetOptions{})
					if err != nil {
						isOrphan = true
						break
					}
				}
			}
		}

		if isOrphan {
			names = append(names, crb.Name)
		}
	}
	return names, nil
}

// isBuiltInClusterRole checks if a cluster role is a built-in Kubernetes role
func isBuiltInClusterRole(name string) bool {
	builtInRoles := map[string]bool{
		"admin":                            true,
		"cluster-admin":                    true,
		"edit":                             true,
		"view":                             true,
		"self-provisioner":                 true,
		"basic-user":                       true,
		"cluster-status":                   true,
		"node-problem-detector":            true,
		"gce:podsecuritypolicy:privileged": true,
	}
	return builtInRoles[name]
}
