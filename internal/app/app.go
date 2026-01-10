package app

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	k8sutil "github.com/kamilbabayev/korp/pkg/k8s"
)

type scanResult struct {
	Namespace                string   `json:"namespace"`
	Pods                     int      `json:"pods"`
	ConfigMaps               int      `json:"configmaps"`
	Secrets                  int      `json:"secrets"`
	Services                 int      `json:"services"`
	PVCs                     int      `json:"pvcs"`
	OrphanConfigMaps         int      `json:"orphan_configmaps"`
	OrphanSecrets            int      `json:"orphan_secrets"`
	OrphanPVCs               int      `json:"orphan_pvcs"`
	ServicesNoEndpoints      int      `json:"services_no_endpoints"`
	OrphanConfigMapNames     []string `json:"orphan_configmap_names,omitempty"`
	OrphanSecretNames        []string `json:"orphan_secret_names,omitempty"`
	OrphanPVCNames           []string `json:"orphan_pvc_names,omitempty"`
	ServicesNoEndpointsNames []string `json:"services_no_endpoints_names,omitempty"`
}

func buildClient(kubeconfig string) (*kubernetes.Clientset, error) {
	// Try in-cluster first when kubeconfig not provided
	if kubeconfig == "" {
		if cfg, err := rest.InClusterConfig(); err == nil {
			return kubernetes.NewForConfig(cfg)
		}
		// fallback to default kubeconfig
		if home, err := os.UserHomeDir(); err == nil {
			kubeconfig = filepath.Join(home, ".kube", "config")
		}
	}

	cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(cfg)
}

// getPodNamespace returns the namespace the pod is running in when running in-cluster.
// Returns empty string if not running in a pod.
func getPodNamespace() string {
	nsPath := "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
	if data, err := os.ReadFile(nsPath); err == nil {
		return strings.TrimSpace(string(data))
	}
	return ""
}

// countIssueTypes returns the number of resource types with issues
func countIssueTypes(res scanResult) int {
	count := 0
	if res.OrphanConfigMaps > 0 {
		count++
	}
	if res.OrphanSecrets > 0 {
		count++
	}
	if res.OrphanPVCs > 0 {
		count++
	}
	if res.ServicesNoEndpoints > 0 {
		count++
	}
	return count
}

// Run performs the main application logic. Supports a simple `scan` command.
func Run(args []string) error {
	fs := flag.NewFlagSet("korp", flag.ContinueOnError)
	namespace := fs.String("namespace", "", "namespace to scan")
	allNamespaces := fs.Bool("all-namespaces", false, "scan all namespaces")
	kubeconfig := fs.String("kubeconfig", "", "path to kubeconfig")
	output := fs.String("output", "table", "output format: table|json")

	if err := fs.Parse(args); err != nil {
		return err
	}

	// Determine target namespace
	ns := *namespace
	if *allNamespaces {
		ns = metav1.NamespaceAll
	} else if ns == "" {
		// Default to scanning all namespaces
		ns = metav1.NamespaceAll
		fmt.Fprintf(os.Stderr, "Scanning all namespaces (use --namespace=<name> to scan specific namespace)\n")
	}

	client, err := buildClient(*kubeconfig)
	if err != nil {
		return fmt.Errorf("building kube client: %w", err)
	}

	ctx := context.TODO()

	pods, err := client.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("listing pods: %w", err)
	}
	cms, err := client.CoreV1().ConfigMaps(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("listing configmaps: %w", err)
	}
	secrets, err := client.CoreV1().Secrets(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("listing secrets: %w", err)
	}
	svcs, err := client.CoreV1().Services(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("listing services: %w", err)
	}
	pvcs, err := client.CoreV1().PersistentVolumeClaims(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("listing pvcs: %w", err)
	}

	res := scanResult{
		Namespace:  ns,
		Pods:       len(pods.Items),
		ConfigMaps: len(cms.Items),
		Secrets:    len(secrets.Items),
		Services:   len(svcs.Items),
		PVCs:       len(pvcs.Items),
	}

	// Detect ownerless (no ownerReferences) items and collect names using helpers
	orphanCMs, err := k8sutil.OrphanConfigMaps(ctx, client, ns)
	if err != nil {
		return fmt.Errorf("finding orphan configmaps: %w", err)
	}
	orphanSecrets, err := k8sutil.OrphanSecrets(ctx, client, ns)
	if err != nil {
		return fmt.Errorf("finding orphan secrets: %w", err)
	}
	orphanPVCs, err := k8sutil.OrphanPVCs(ctx, client, ns)
	if err != nil {
		return fmt.Errorf("finding orphan pvcs: %w", err)
	}
	svcsNoEP, err := k8sutil.ServicesWithoutEndpoints(ctx, client, ns)
	if err != nil {
		return fmt.Errorf("finding services without endpoints: %w", err)
	}

	res.OrphanConfigMapNames = orphanCMs
	res.OrphanSecretNames = orphanSecrets
	res.OrphanPVCNames = orphanPVCs
	res.ServicesNoEndpointsNames = svcsNoEP

	res.OrphanConfigMaps = len(orphanCMs)
	res.OrphanSecrets = len(orphanSecrets)
	res.OrphanPVCs = len(orphanPVCs)
	res.ServicesNoEndpoints = len(svcsNoEP)

	switch *output {
	case "json":
		b, _ := json.MarshalIndent(res, "", "  ")
		fmt.Println(string(b))
	default:
		// Print header
		fmt.Println("================================================================================")
		fmt.Println("KORP SCAN RESULTS")
		fmt.Println("================================================================================")

		// Show namespace info
		nsDisplay := res.Namespace
		if res.Namespace == "" || res.Namespace == metav1.NamespaceAll {
			nsDisplay = "All Namespaces"
		}
		fmt.Printf("\nTarget: %s\n\n", nsDisplay)

		// Resource summary
		fmt.Println("RESOURCE SUMMARY:")
		fmt.Println("--------------------------------------------------------------------------------")
		fmt.Printf("  Pods:         %d\n", res.Pods)
		fmt.Printf("  ConfigMaps:   %d\n", res.ConfigMaps)
		fmt.Printf("  Secrets:      %d\n", res.Secrets)
		fmt.Printf("  Services:     %d\n", res.Services)
		fmt.Printf("  PVCs:         %d\n", res.PVCs)

		// Orphaned resources with inline details
		fmt.Println("\nORPHANED RESOURCES:")
		fmt.Println("================================================================================")

		hasFindings := false

		// Orphaned ConfigMaps
		if res.OrphanConfigMaps > 0 {
			hasFindings = true
			fmt.Printf("\nConfigMaps: %d orphaned\n", res.OrphanConfigMaps)
			for i, name := range res.OrphanConfigMapNames {
				fmt.Printf("   %d. %s\n", i+1, name)
			}
		} else {
			fmt.Printf("\nConfigMaps: No orphaned resources\n")
		}

		// Orphaned Secrets
		if res.OrphanSecrets > 0 {
			hasFindings = true
			fmt.Printf("\nSecrets: %d orphaned\n", res.OrphanSecrets)
			for i, name := range res.OrphanSecretNames {
				fmt.Printf("   %d. %s\n", i+1, name)
			}
		} else {
			fmt.Printf("\nSecrets: No orphaned resources\n")
		}

		// Orphaned PVCs
		if res.OrphanPVCs > 0 {
			hasFindings = true
			fmt.Printf("\nPVCs: %d orphaned\n", res.OrphanPVCs)
			for i, name := range res.OrphanPVCNames {
				fmt.Printf("   %d. %s\n", i+1, name)
			}
		} else {
			fmt.Printf("\nPVCs: No orphaned resources\n")
		}

		// Services without endpoints
		if res.ServicesNoEndpoints > 0 {
			hasFindings = true
			fmt.Printf("\nServices: %d without endpoints\n", res.ServicesNoEndpoints)
			for i, name := range res.ServicesNoEndpointsNames {
				fmt.Printf("   %d. %s\n", i+1, name)
			}
		} else {
			fmt.Printf("\nServices: All have endpoints\n")
		}

		// Footer
		fmt.Println("\n================================================================================")
		if hasFindings {
			fmt.Printf("Found issues in %d resource type(s)\n", countIssueTypes(res))
		} else {
			fmt.Println("No orphaned resources found - cluster is clean!")
		}
		fmt.Println("================================================================================")
	}

	return nil
}
