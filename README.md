# Korp - Kubernetes Orphaned Resource Pointer

Korp is both a CLI tool and Kubernetes operator for detecting and reporting orphaned Kubernetes resources. Orphaned resources are objects that lack proper ownership references or have missing dependencies, which can lead to resource leaks and cluster sprawl.

## Features

- **Orphan Detection**: Identifies orphaned resources across 16 resource types:
  - ConfigMaps, Secrets, PVCs (without owner references)
  - Services (without active Endpoints)
  - Deployments, StatefulSets, DaemonSets (scaled to zero or no ready pods)
  - Jobs, CronJobs (completed/suspended)
  - ReplicaSets (orphaned from deleted Deployments)
  - ServiceAccounts (not used by any pod)
  - Ingresses (pointing to non-existent services)
  - Roles, ClusterRoles (not referenced by any binding)
  - RoleBindings, ClusterRoleBindings (referencing non-existent roles/subjects)
- **Auto-Cleanup**: Safely remove orphaned resources with dry-run mode, age thresholds, and preservation labels
- **Flexible Filtering**: Exclude resources by name patterns or labels
- **Dual Mode**: Run as CLI tool or Kubernetes operator
- **Event Reporting**: Creates Kubernetes events for findings
- **Historical Tracking**: Maintains scan history and trends
- **Webhook Notifications**: Send scan results to external systems

## Quick Start

### CLI Mode

#### Build
```bash
make build-cli
```

#### Usage
```bash
# Default: scan all namespaces
./bin/korp

# Scan a specific namespace
./bin/korp --namespace default

# Scan all namespaces (explicit)
./bin/korp --all-namespaces

# JSON output for all namespaces
./bin/korp --output json

# JSON output for specific namespace
./bin/korp --namespace default --output json
```

#### Run as Kubernetes Pod

You can run the CLI directly in your cluster using `kubectl run`:

```bash
# Default: scan all namespaces
kubectl run korp-scan --rm -i --restart=Never --image=kamilbabayev/korp-cli:latest -n korp

# Scan a specific namespace
kubectl run korp-scan --rm -i --restart=Never --image=kamilbabayev/korp-cli:latest -n korp -- --namespace=production

# Scan all namespaces (explicit)
kubectl run korp-scan --rm -i --restart=Never --image=kamilbabayev/korp-cli:latest -n korp -- --all-namespaces

# JSON output for all namespaces
kubectl run korp-scan --rm -i --restart=Never --image=kamilbabayev/korp-cli:latest -n korp -- --output=json

# JSON output for specific namespace
kubectl run korp-scan --rm -i --restart=Never --image=kamilbabayev/korp-cli:latest -n korp -- --namespace=production --output=json
```

**Note**: When running without arguments, korp scans **all namespaces** by default. Use `--namespace=<name>` to scan a specific namespace.

### Operator Mode

#### Installation

**Option 1: Using Helm from Public Repository (Recommended)**

```bash
# Add the Helm repository
helm repo add korp https://kamilbabayev.github.io/korp

# Update your local Helm chart repository cache
helm repo update

# Install the operator
helm install korp korp/korp --namespace korp --create-namespace
```

**Option 2: Using Helm from Source**

```bash
# Clone the repository first
git clone https://github.com/kamilbabayev/korp.git
cd korp

# Install the operator
helm install korp ./charts/korp --namespace korp --create-namespace

# Or using make
make helm-install
```

**Option 3: Using kubectl**

```bash
# Install CRD
make install

# Deploy Operator
make deploy

# Create a KorpScan Resource
kubectl apply -f config/samples/basic_scan.yaml
```

#### Uninstall

**Using Helm:**
```bash
helm uninstall korp --namespace korp
# Or
make helm-uninstall
```

**Using kubectl:**
```bash
make undeploy
make uninstall
```

## Operator Usage

### Basic Scan Example

```yaml
apiVersion: korp.io/v1alpha1
kind: KorpScan
metadata:
  name: default-namespace-scan
  namespace: korp
spec:
  targetNamespace: "default"
  intervalMinutes: 30
  resourceTypes:
    - configmaps
    - secrets
    - pvcs
    - services
    - deployments
    - statefulsets
    - daemonsets
    - jobs
    - cronjobs
    - replicasets
    - serviceaccounts
    - ingresses
  reporting:
    createEvents: true
    eventSeverity: "Warning"
    historyLimit: 5
```

### Cluster-Wide Scan

```yaml
apiVersion: korp.io/v1alpha1
kind: KorpScan
metadata:
  name: cluster-scan
  namespace: korp
spec:
  targetNamespace: "*"
  intervalMinutes: 120
  resourceTypes:
    - configmaps
    - secrets
```

### Filtered Scan

```yaml
apiVersion: korp.io/v1alpha1
kind: KorpScan
metadata:
  name: production-scan
  namespace: korp
spec:
  targetNamespace: "production"
  intervalMinutes: 15
  filters:
    excludeNamePatterns:
      - "^default-token-.*"
      - "^sh\\.helm\\..*"
  reporting:
    createEvents: true
    eventSeverity: "Warning"
```

### Scan with Auto-Cleanup (Dry-Run)

```yaml
apiVersion: korp.io/v1alpha1
kind: KorpScan
metadata:
  name: cleanup-scan
  namespace: korp
spec:
  targetNamespace: "staging"
  intervalMinutes: 60
  resourceTypes:
    - configmaps
    - secrets
    - jobs
  cleanup:
    enabled: true
    dryRun: true              # Safe: only logs what would be deleted
    minAgeDays: 14            # Only cleanup resources orphaned for 14+ days
    preservationLabels:
      - "korp.io/preserve"    # Resources with this label are never deleted
      - "do-not-delete"
  reporting:
    createEvents: true
```

### Scan with Auto-Cleanup (Active)

> **Warning**: Set `dryRun: false` only after verifying dry-run results.

```yaml
apiVersion: korp.io/v1alpha1
kind: KorpScan
metadata:
  name: active-cleanup-scan
  namespace: korp
spec:
  targetNamespace: "dev"
  intervalMinutes: 120
  resourceTypes:
    - configmaps
    - secrets
    - jobs
    - replicasets
  cleanup:
    enabled: true
    dryRun: false             # Actually deletes resources!
    minAgeDays: 30            # Conservative: 30 days minimum
    resourceTypes:            # Only cleanup specific types
      - configmaps
      - jobs
    preservationLabels:
      - "korp.io/preserve"
      - "important"
  reporting:
    createEvents: true
```

## KorpScan CRD Reference

### Spec Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `targetNamespace` | string | Yes | - | Namespace to scan. Use "*" for all namespaces |
| `intervalMinutes` | int | No | 60 | Scan interval in minutes |
| `resourceTypes` | []string | No | all | Resource types to scan (see below) |
| `filters.excludeNamePatterns` | []string | No | [] | Regex patterns to exclude resources by name |
| `filters.excludeLabels` | map[string]string | No | {} | Label selectors to exclude resources |
| `reporting.createEvents` | bool | No | true | Whether to create Kubernetes events |
| `reporting.eventSeverity` | string | No | Warning | Event severity: Normal or Warning |
| `reporting.historyLimit` | int | No | 5 | Number of scan results to retain |
| `cleanup.enabled` | bool | No | false | Enable automatic cleanup of orphaned resources |
| `cleanup.dryRun` | bool | No | true | If true, only log what would be deleted (safe mode) |
| `cleanup.minAgeDays` | int | No | 7 | Minimum days a resource must be orphaned before cleanup |
| `cleanup.resourceTypes` | []string | No | all | Specific resource types to cleanup |
| `cleanup.preservationLabels` | []string | No | [] | Labels that prevent cleanup when present |

### Supported Resource Types

| Type | Description | Orphan Detection |
|------|-------------|------------------|
| `configmaps` | ConfigMaps | No owner reference and not used by pods |
| `secrets` | Secrets | No owner reference and not used by pods |
| `pvcs` | PersistentVolumeClaims | No owner reference and not mounted |
| `services` | Services | No active endpoints |
| `deployments` | Deployments | Scaled to zero or no ready pods |
| `statefulsets` | StatefulSets | Scaled to zero or no ready pods |
| `daemonsets` | DaemonSets | No scheduled or ready pods |
| `jobs` | Jobs | Completed and older than 7 days |
| `cronjobs` | CronJobs | Suspended with no recent success |
| `replicasets` | ReplicaSets | No owner reference and zero replicas |
| `serviceaccounts` | ServiceAccounts | Not used by any pod |
| `ingresses` | Ingresses | Backend service doesn't exist |
| `roles` | Roles | Not referenced by any RoleBinding |
| `clusterroles` | ClusterRoles | Not referenced by any binding |
| `rolebindings` | RoleBindings | References non-existent Role or ServiceAccount |
| `clusterrolebindings` | ClusterRoleBindings | References non-existent ClusterRole or ServiceAccount |

### Status Fields

| Field | Description |
|-------|-------------|
| `phase` | Current scan state: Pending, Running, Completed, Failed |
| `lastScanTime` | Timestamp of last completed scan |
| `summary.orphanedConfigMaps` | Count of orphaned ConfigMaps |
| `summary.orphanedSecrets` | Count of orphaned Secrets |
| `summary.orphanedPVCs` | Count of orphaned PVCs |
| `summary.servicesWithoutEndpoints` | Count of Services without Endpoints |
| `summary.orphanedDeployments` | Count of orphaned Deployments |
| `summary.orphanedStatefulSets` | Count of orphaned StatefulSets |
| `summary.orphanedDaemonSets` | Count of orphaned DaemonSets |
| `summary.orphanedJobs` | Count of orphaned Jobs |
| `summary.orphanedCronJobs` | Count of orphaned CronJobs |
| `summary.orphanedReplicaSets` | Count of orphaned ReplicaSets |
| `summary.orphanedServiceAccounts` | Count of orphaned ServiceAccounts |
| `summary.orphanedIngresses` | Count of orphaned Ingresses |
| `summary.orphanedRoles` | Count of orphaned Roles |
| `summary.orphanedClusterRoles` | Count of orphaned ClusterRoles |
| `summary.orphanedRoleBindings` | Count of orphaned RoleBindings |
| `summary.orphanedClusterRoleBindings` | Count of orphaned ClusterRoleBindings |
| `summary.orphanCount` | Total count of all orphaned resources |
| `findings` | Detailed list of orphaned resources |
| `history` | Recent scan results with timestamps and counts |
| `conditions` | Standard Kubernetes conditions |
| `cleanupStatus.lastCleanupTime` | Timestamp of last cleanup operation |
| `cleanupStatus.lastCleanupResult` | Result: Success, DryRun, PartialFailure |
| `cleanupStatus.summary` | Cleanup counts (deleted, failed, skipped) |

## Viewing Results

### Check KorpScan Status
```bash
kubectl get korpscans -A
kubectl describe korpscan default-namespace-scan -n korp
```

### View Findings
```bash
kubectl get korpscan default-namespace-scan -n korp -o jsonpath='{.status.findings}' | jq
```

### View Events on Orphaned Resources
```bash
# View all orphan events cluster-wide (recommended)
kubectl get events -A --field-selector reason=Orphaned

# View events related to KorpScan resource
kubectl get events -n korp --field-selector involvedObject.kind=KorpScan
```

## Development

### Prerequisites
- Go 1.20+
- Kubernetes cluster (for operator mode)
- kubectl
- Docker (for building images)

### Build

```bash
# Build operator
make build

# Build CLI
make build-cli

# Build Docker image
make docker-build IMG=your-registry/korp-operator:tag
```

### Testing

```bash
# Run tests
make test

# Run operator locally (requires kubeconfig)
make run
```

### Code Generation

```bash
# Generate CRDs
make manifests

# Generate deepcopy code
make generate
```

## Architecture

```
korp/
├── api/v1alpha1/          # CRD types
├── cmd/
│   ├── cli/              # CLI entry point
│   └── operator/         # Operator binary entry point
├── config/                # Kubernetes manifests
│   ├── crd/              # CRD definitions
│   ├── rbac/             # RBAC rules
│   ├── operator/         # Operator deployment
│   └── samples/          # Example KorpScans
├── charts/korp/          # Helm chart
├── internal/
│   ├── app/              # CLI logic
│   └── controller/       # Operator controller
└── pkg/
    ├── k8s/              # K8s detection utilities
    ├── scan/             # Scan orchestration
    ├── cleanup/          # Auto-cleanup logic
    ├── notifier/         # Webhook notifications
    └── reporter/         # Event reporting
```

## RBAC Permissions

The operator requires the following permissions:

- **Read**: Pods, Endpoints (for usage detection)
- **Read/Delete**: ConfigMaps, Secrets, PVCs, Services, ServiceAccounts, Deployments, StatefulSets, DaemonSets, ReplicaSets, Jobs, CronJobs, Ingresses
- **Write**: Events
- **Full**: KorpScan custom resources, Leases (leader election)

## Troubleshooting

### Operator not starting
```bash
kubectl logs -n korp deployment/korp-operator
```

### Scans not running
Check the KorpScan status:
```bash
kubectl describe korpscan <name> -n korp
```

### No events created
Verify `spec.reporting.createEvents: true` in your KorpScan

## Contributing

Contributions welcome! Please:
1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Submit a pull request

## License

MIT License - See LICENSE file for details

## Roadmap

- [x] ~~Auto-cleanup mode (with safety controls)~~ - **Implemented!**
- [x] ~~Webhook notifications~~ - **Implemented!**
- [ ] Prometheus metrics export
- [ ] Multi-cluster support
- [ ] Slack/email notifications (native integration)
- [ ] Custom policy definitions
- [ ] Webhooks for CRD validation
