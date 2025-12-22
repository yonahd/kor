# Release Process

This document describes the release process for the Kor project.

## Overview

Kor has two separate release workflows:

1. **Application Release** - Releases the Kor application binary, Docker images, and Krew plugin
2. **Helm Chart Release** - Releases the Helm chart independently

## Application Release

Application releases are triggered by pushing a tag with the format `v*` (e.g., `v0.6.7`, `v1.0.0`).

### Steps to Release the Application

1. Update the application code and ensure all tests pass
2. Create and push a tag with the `v*` format:
   ```bash
   git tag v0.6.7
   git push origin v0.6.7
   ```

3. The following will be automatically released:
   - GitHub release with application binaries (via GoReleaser)
   - Docker images for multiple platforms
   - Krew plugin update
   - **Helm chart** (if Chart.yaml version was updated)

### Workflow File

See `.github/workflows/release.yml`

## Helm Chart Release

Helm chart releases can be done independently of application releases. This is useful when:
- Fixing bugs in chart templates
- Updating chart documentation
- Modifying chart values or configurations
- Any chart-only changes that don't require an application update

### Steps to Release the Chart Only

1. Update the chart version in `charts/kor/Chart.yaml`:
   ```yaml
   version: 0.2.10  # Increment the version
   appVersion: "0.6.6"  # Keep the same app version
   ```

2. Commit the changes to the main branch:
   ```bash
   git add charts/kor/Chart.yaml
   git commit -m "chore: bump chart version to 0.2.10"
   git push origin main
   ```

3. Create and push a tag with the format `kor-*` matching the new chart version:
   ```bash
   git tag kor-0.2.10
   git push origin kor-0.2.10
   ```

4. The Helm chart will be automatically released to the `gh-pages` branch and made available via:
   ```bash
   helm repo add kor https://yonahd.github.io/kor
   helm repo update
   helm install kor kor/kor --version 0.2.10
   ```

### Workflow File

See `.github/workflows/chart-release.yml`

## Chart Release Triggers

The chart release workflow is triggered by tags matching:
- `v*` - Application release tags (maintains backward compatibility)
- `kor-*` - Chart-only release tags

This ensures that:
1. Every application release also releases a chart (if the chart version changed)
2. Charts can be released independently without requiring an application release

## Best Practices

1. **Semantic Versioning**: Follow [SemVer](https://semver.org/) for both application and chart versions
2. **Chart Version Increments**:
   - Patch version: Bug fixes, documentation updates
   - Minor version: New features, non-breaking changes
   - Major version: Breaking changes
3. **Application Version Alignment**: Update `appVersion` in Chart.yaml when releasing both application and chart together
4. **Testing**: Run chart tests locally before releasing:
   ```bash
   helm lint charts/kor
   helm template charts/kor
   ```

## Troubleshooting

### Chart Release Not Created

If a chart release is not created after pushing a `kor-*` tag:
1. Verify the chart version in `Chart.yaml` matches the tag (e.g., `kor-0.2.10` requires `version: 0.2.10`)
2. Check the GitHub Actions workflow logs
3. Ensure the chart version is higher than any existing released version

### Application Release Also Releasing Chart

This is expected behavior. Application releases (`v*` tags) will also trigger the chart release workflow to maintain backward compatibility.
