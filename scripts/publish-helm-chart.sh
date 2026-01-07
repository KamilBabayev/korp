#!/bin/bash
set -e

CHART_DIR="charts/korp-operator"
DIST_DIR="dist"
REPO_URL="https://kamilbabayev.github.io/korp"

echo "==> Publishing Helm Chart"

# Check if helm is installed
if ! command -v helm &> /dev/null; then
    echo "Error: helm is not installed. Please install Helm first."
    exit 1
fi

# Check if we're in the git repository root
if [ ! -f "Chart.yaml" ] && [ ! -d "charts/korp-operator" ]; then
    echo "Error: Must run from repository root"
    exit 1
fi

# Get chart version
CHART_VERSION=$(grep '^version:' ${CHART_DIR}/Chart.yaml | awk '{print $2}')
echo "Chart version: ${CHART_VERSION}"

# Create dist directory if it doesn't exist
mkdir -p ${DIST_DIR}

# Package the chart
echo "==> Packaging chart..."
helm package ${CHART_DIR} -d ${DIST_DIR}

# Check if gh-pages branch exists
if git show-ref --verify --quiet refs/heads/gh-pages; then
    echo "==> gh-pages branch exists"
else
    echo "==> Creating gh-pages branch..."
    git checkout --orphan gh-pages
    git rm -rf .
    git commit --allow-empty -m "Initial gh-pages commit"
    git push -u origin gh-pages
    git checkout main
fi

# Save current branch
CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)

# Switch to gh-pages branch
echo "==> Switching to gh-pages branch..."
git checkout gh-pages

# Copy packaged chart
cp ${DIST_DIR}/korp-operator-${CHART_VERSION}.tgz .

# Update or create index
echo "==> Updating Helm repository index..."
helm repo index . --url ${REPO_URL}

# Commit and push
echo "==> Committing changes..."
git add korp-operator-${CHART_VERSION}.tgz index.yaml
git commit -m "Release korp-operator chart version ${CHART_VERSION}"

echo "==> Pushing to gh-pages branch..."
git push origin gh-pages

# Switch back to original branch
git checkout ${CURRENT_BRANCH}

# Clean up dist directory
rm -rf ${DIST_DIR}

echo ""
echo "==> Chart published successfully!"
echo ""
echo "Users can now install with:"
echo "  helm repo add korp ${REPO_URL}"
echo "  helm repo update"
echo "  helm install korp-operator korp/korp-operator"
echo ""
echo "Verify at: ${REPO_URL}/index.yaml"
