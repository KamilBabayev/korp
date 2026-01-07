# Publishing Helm Chart to GitHub Pages

This document explains how the Helm chart is automatically published and how users can install it.

## Automatic Publishing

The Helm chart is automatically published to GitHub Pages when changes are pushed to the `main` branch.

### How it Works

1. **GitHub Actions Workflow**: `.github/workflows/release-helm-chart.yaml` triggers on pushes to `main` that affect the chart
2. **Chart Releaser**: Uses `helm/chart-releaser-action` to:
   - Package the Helm chart
   - Create a GitHub Release
   - Update the Helm repository index (`index.yaml`)
   - Publish to GitHub Pages

### First-Time Setup (Already Done)

These steps were completed during initial setup:

1. ✅ Created `.github/workflows/release-helm-chart.yaml`
2. ✅ Enabled GitHub Pages in repository settings
   - Go to Settings → Pages
   - Source: Deploy from a branch
   - Branch: `gh-pages` / `root`

## For Users: Installing the Chart

Once published, users can install without cloning the repository:

```bash
# Add the Helm repository
helm repo add korp https://kamilbabayev.github.io/korp

# Update repository cache
helm repo update

# Install the operator
helm install korp-operator korp/korp-operator \
  --namespace korp-operator \
  --create-namespace

# Search for available versions
helm search repo korp
```

## For Maintainers: Publishing a New Version

1. Update the chart version in `charts/korp-operator/Chart.yaml`:
   ```yaml
   version: 0.2.0  # Increment this
   ```

2. Commit and push to main:
   ```bash
   git add charts/korp-operator/Chart.yaml
   git commit -m "Bump chart version to 0.2.0"
   git push origin main
   ```

3. GitHub Actions will automatically:
   - Package the chart
   - Create a GitHub Release (v0.2.0)
   - Update the Helm repository index
   - Publish to GitHub Pages

## Verifying Publication

After pushing, check:

1. **GitHub Actions**: https://github.com/kamilbabayev/korp/actions
   - Verify "Release Helm Chart" workflow succeeded

2. **GitHub Releases**: https://github.com/kamilbabayev/korp/releases
   - New release should appear with chart package

3. **GitHub Pages**: https://kamilbabayev.github.io/korp/index.yaml
   - Should show updated chart version

4. **Test Installation**:
   ```bash
   helm repo add korp https://kamilbabayev.github.io/korp
   helm repo update
   helm search repo korp --versions
   ```

## Chart Repository Structure

After publishing, GitHub Pages will host:

```
https://kamilbabayev.github.io/korp/
├── index.yaml                      # Helm repository index
└── korp-operator-0.1.0.tgz        # Packaged chart (in releases)
```

## Troubleshooting

### Chart not appearing after push

1. Check GitHub Actions logs for errors
2. Verify GitHub Pages is enabled
3. Ensure `gh-pages` branch exists
4. Wait a few minutes for GitHub Pages to update

### Version conflicts

- Chart versions must be unique
- Increment version in `Chart.yaml` before each release
- Follow semantic versioning (MAJOR.MINOR.PATCH)

### Manual Publishing (if needed)

If automatic publishing fails:

```bash
# Package the chart
helm package charts/korp-operator -d dist/

# Create/update index
helm repo index dist/ --url https://kamilbabayev.github.io/korp

# Commit to gh-pages branch
git checkout gh-pages
cp dist/* .
git add .
git commit -m "Release chart version X.Y.Z"
git push origin gh-pages
```
