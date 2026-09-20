# R8 Direct Provider Release

Date: 2026-09-20
Lane: CPA/Vision
Repository: `D:/Python/projects/CPA Plugin/agy-identity-bridge`
Branch: `main`

## Release

- Version: `0.4.0`
- Build host: `10.21.1.101` (`alpine-docker`)
- Build runtime: Docker `golang:1.26-bookworm`
- Build mode: `CGO_ENABLED=1`, `-buildmode=c-shared`, version ldflag `0.4.0`
- Installed artifact:
  `/home/Docker/CLIProxyAPI/plugins/linux/amd64/any2api-bridge-v0.4.0.so`
- Mode: `0755`
- SHA-256:
  `a1b72cc16304da9b7f6fbb594d32ee0043eada5e5f9efec65527e3f369c5af72`
- Previous artifact moved outside the plugin directory:
  `/home/Docker/CLIProxyAPI/plugin-backups/any2api-bridge-v0.3.0.so`

## CPA Config Pin

In `/home/Docker/CLIProxyAPI/config.yaml`:

- `plugins.configs.any2api-bridge.store.version: 0.4.0`
- `plugins.configs.any2api-bridge.store.release-tag: v0.4.0`
- Backup created at `config.yaml.bak-0.4.0`

## Verification

- Exactly one `any2api-bridge-*.so` is present in the plugin directory.
- The running `cli-proxy-api` container sees the new file through the existing
  bind mount.
- The container was not restarted in this round.

## Boundaries

- No secrets or raw headers are included in this report.
- No CPA restart was performed.
