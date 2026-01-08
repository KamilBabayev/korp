# Korp Operator Helm Chart

This Helm chart deploys the Korp operator for detecting and reporting orphaned Kubernetes resources.

## Prerequisites

- Kubernetes 1.19+
- Helm 3.0+

## Installing the Chart

### From Public Helm Repository (Recommended)

```bash
# Add the Helm repository
helm repo add korp https://kamilbabayev.github.io/korp

# Update your local Helm chart repository cache
helm repo update

# Install the chart
helm install korp korp/korp --namespace korp-operator --create-namespace
```

### From Source

To install the chart with the release name `korp`:

```bash
helm install korp ./charts/korp
```

To install in a specific namespace:

```bash
helm install korp ./charts/korp --namespace korp-operator --create-namespace
```

## Uninstalling the Chart

To uninstall/delete the `korp` deployment:

```bash
helm uninstall korp --namespace korp-operator
```

## Configuration

The following table lists the configurable parameters of the Korp Operator chart and their default values.

| Parameter | Description | Default |
|-----------|-------------|---------|
| `namespaceOverride` | Override the namespace | `""` |
| `image.repository` | Image repository | `korp-operator` |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `image.tag` | Image tag | `latest` |
| `imagePullSecrets` | Image pull secrets | `[]` |
| `replicaCount` | Number of replicas | `1` |
| `serviceAccount.create` | Create service account | `true` |
| `serviceAccount.name` | Service account name | `korp-operator` |
| `serviceAccount.annotations` | Service account annotations | `{}` |
| `rbac.create` | Create RBAC resources | `true` |
| `leaderElection.enabled` | Enable leader election | `true` |
| `resources.limits.cpu` | CPU limit | `200m` |
| `resources.limits.memory` | Memory limit | `128Mi` |
| `resources.requests.cpu` | CPU request | `100m` |
| `resources.requests.memory` | Memory request | `64Mi` |
| `nodeSelector` | Node selector | `{}` |
| `tolerations` | Tolerations | `[]` |
| `affinity` | Affinity | `{}` |
| `podAnnotations` | Pod annotations | `{}` |
| `podSecurityContext` | Pod security context | See values.yaml |
| `securityContext` | Container security context | See values.yaml |
| `livenessProbe` | Liveness probe configuration | See values.yaml |
| `readinessProbe` | Readiness probe configuration | See values.yaml |
| `metrics.enabled` | Enable metrics endpoint | `true` |
| `metrics.port` | Metrics port | `8080` |
| `healthProbe.enabled` | Enable health probe endpoint | `true` |
| `healthProbe.port` | Health probe port | `8081` |

## Example: Custom Values

Create a `custom-values.yaml` file:

```yaml
image:
  repository: myregistry/korp-operator
  tag: "v0.1.0"

resources:
  limits:
    cpu: 500m
    memory: 256Mi
  requests:
    cpu: 250m
    memory: 128Mi

nodeSelector:
  node-role.kubernetes.io/control-plane: ""

tolerations:
  - key: node-role.kubernetes.io/control-plane
    operator: Exists
    effect: NoSchedule
```

Install with custom values:

```bash
helm install korp ./charts/korp -f custom-values.yaml
```

## Example: Creating a KorpScan

After installing the operator, create a KorpScan resource:

```yaml
apiVersion: korp.io/v1alpha1
kind: KorpScan
metadata:
  name: default-scan
  namespace: korp-operator
spec:
  targetNamespace: "default"
  intervalMinutes: 30
  resourceTypes:
    - configmaps
    - secrets
  filters:
    excludeNamePatterns:
      - "^default-token-.*"
  reporting:
    createEvents: true
    eventSeverity: "Warning"
```

## Upgrading

To upgrade the operator:

```bash
helm upgrade korp ./charts/korp
```

## Development

To lint the chart:

```bash
helm lint ./charts/korp
```

To template the chart (dry-run):

```bash
helm template korp ./charts/korp
```

To package the chart:

```bash
helm package ./charts/korp
```
