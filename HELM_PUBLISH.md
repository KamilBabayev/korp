# Publishing Helm Chart to GitHub Pages

This document explains how to manually publish the Helm chart to GitHub Pages so users can install it without cloning the repository.

## Prerequisites

- Helm 3.x installed
- Git configured with push access to the repository
- GitHub Pages enabled in repository settings

## First-Time Setup

### 1. Enable GitHub Pages

1. Go to repository Settings → Pages
2. Source: Deploy from a branch
3. Branch: `gh-pages` / `root`
4. Click Save

The script will create the `gh-pages` branch automatically if it doesn't exist.

## Publishing a New Chart Version

### 1. Update Chart Version

Edit `charts/korp/Chart.yaml` and increment the version:

```yaml
version: 0.2.0  # Increment this
appVersion: "0.2.0"  # Update if application version changed
```

### 2. Run the Publish Script

From the repository root:

```bash
./scripts/publish-helm-chart.sh
```

Or using Make:

```bash
make helm-publish
```

The script will:
- Package the Helm chart
- Create/update the `gh-pages` branch
- Generate/update the Helm repository index (`index.yaml`)
- Commit and push to GitHub Pages

### 3. Verify Publication

After a few minutes, verify the chart is available:

```bash
# Check the index file
curl https://kamilbabayev.github.io/korp/index.yaml

# Test installation
helm repo add korp https://kamilbabayev.github.io/korp
helm repo update
helm search repo korp --versions
```

## For Users: Installing the Chart

Once published, users can install without cloning the repository:

```bash
# Add the Helm repository
helm repo add korp https://kamilbabayev.github.io/korp

# Update repository cache
helm repo update

# Install the operator
helm install korp korp/korp \
  --namespace korp \
  --create-namespace

# Search for available versions
helm search repo korp

# Install specific version
helm install korp korp/korp \
  --version 0.1.0 \
  --namespace korp \
  --create-namespace
```

## Chart Repository Structure

After publishing, GitHub Pages hosts:

```
https://kamilbabayev.github.io/korp/
├── index.yaml                # Helm repository index
├── korp-0.1.0.tgz           # Packaged chart v0.1.0
└── korp-0.2.0.tgz           # Packaged chart v0.2.0
```

## Troubleshooting

### Chart not appearing after publishing

1. Wait 5-10 minutes for GitHub Pages to update
2. Check GitHub Pages is enabled in Settings
3. Verify `gh-pages` branch exists and has commits
4. Check repository visibility (public repositories work immediately)

### Version conflicts

- Chart versions must be unique
- Increment version in `Chart.yaml` before publishing
- Follow semantic versioning (MAJOR.MINOR.PATCH)
- Publishing the same version twice will overwrite the previous package

### Permission errors

- Ensure you have push access to the repository
- Check GitHub authentication is configured correctly
- If using SSH, verify SSH keys are set up

### Script fails on gh-pages branch

If the script fails while on the `gh-pages` branch:

```bash
# Switch back to main branch
git checkout main

# Clean up any uncommitted changes on gh-pages
git checkout gh-pages
git reset --hard origin/gh-pages
git checkout main
```

## Manual Publishing (Alternative)

If you prefer not to use the script:

```bash
# 1. Package the chart
helm package charts/korp -d dist/

# 2. Switch to gh-pages branch
git checkout gh-pages

# 3. Copy the package
cp dist/korp-*.tgz .

# 4. Update the index
helm repo index . --url https://kamilbabayev.github.io/korp

# 5. Commit and push
git add *.tgz index.yaml
git commit -m "Release chart version X.Y.Z"
git push origin gh-pages

# 6. Switch back to main
git checkout main
```

## Best Practices

1. **Version Bumping**: Always increment the chart version before publishing
2. **Testing**: Test the chart locally with `helm install` before publishing
3. **Changelog**: Document changes in the chart's README.md
4. **AppVersion**: Update `appVersion` when the application version changes
5. **Dependencies**: Run `helm dependency update` if chart has dependencies
