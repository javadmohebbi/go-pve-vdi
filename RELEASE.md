# Release Process

This document describes how to create a new release of go-pve-vdi.

## Automated Releases (Recommended)

Releases are automatically built and published via GitHub Actions when you push a version tag.

### Steps:

1. **Commit all changes:**
   ```bash
   git add .
   git commit -m "Prepare for release v1.0.0"
   ```

2. **Create and push a version tag:**
   ```bash
   git tag -a v1.0.0 -m "Release version 1.0.0"
   git push origin main
   git push origin v1.0.0
   ```

3. **GitHub Actions will automatically:**
   - Build binaries for all platforms (Linux, macOS, Windows)
   - Generate SHA256 checksums
   - Create a GitHub Release
   - Upload all binaries and checksums

4. **Users can download binaries from:**
   ```
   https://github.com/javadmohebbi/go-pve-vdi/releases
   ```

## Manual Release

If you want to build releases locally:

```bash
# Build all platforms
make release

# Binaries will be in build/ directory
ls -lh build/

# Upload to GitHub manually or use gh CLI:
gh release create v1.0.0 build/* --title "Release v1.0.0" --notes "Release notes here"
```

## Supported Platforms

The following binaries are automatically built:

- **Linux:**
  - `go-pve-vdi-linux-amd64` (64-bit Intel/AMD)
  - `go-pve-vdi-linux-386` (32-bit Intel/AMD)
  - `go-pve-vdi-linux-arm64` (64-bit ARM - Raspberry Pi 4, etc.)
  - `go-pve-vdi-linux-arm` (32-bit ARM - Raspberry Pi 3, etc.)

- **macOS:**
  - `go-pve-vdi-darwin-amd64` (Intel Macs)
  - `go-pve-vdi-darwin-arm64` (Apple Silicon M1/M2/M3)

- **Windows:**
  - `go-pve-vdi-windows-amd64.exe` (64-bit)
  - `go-pve-vdi-windows-386.exe` (32-bit)

## Version Numbering

Follow [Semantic Versioning](https://semver.org/):
- `v1.0.0` - Major release
- `v1.1.0` - Minor release (new features, backwards compatible)
- `v1.1.1` - Patch release (bug fixes)

## Checklist Before Release

- [ ] All tests pass
- [ ] Documentation is up to date
- [ ] CHANGELOG.md is updated
- [ ] Version tag follows semver
- [ ] No sensitive data in code or configs
