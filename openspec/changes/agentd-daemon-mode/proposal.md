## Why

When SSH drops, agentd dies — all running commands, shell state, and jobs are lost. The agent must manually reconnect and restart everything. This is because agentd runs as a child of the SSH process, not as an independent daemon. Additionally, all commands are blocking — long-running tasks (training, installs) freeze the agent until completion with no way to check status or do other work.

## What Changes

- agentd becomes a persistent daemon (systemd user service) listening on a Unix socket (`/run/user/<uid>/agentd.sock`)
- SSH becomes just a tunnel to the socket, not the lifecycle manager — SSH drops no longer kill agentd
- Add `agentctl run --bg` for non-blocking command execution (returns job ID)
- Add `agentctl jobs`, `agentctl job <id>`, `agentctl kill <id>` CLI commands
- Add install script (`install.sh` on GitHub) — on first connect, prompt user "agentd not installed. Install? [y/n]"
- Unix socket permissions (0700) provide auth — no tokens needed

## Capabilities

### New Capabilities
- `daemon-lifecycle`: agentd runs as a systemd user service, survives SSH drops, auto-starts on boot. Install via curl script from GitHub.
- `socket-transport`: agentd listens on Unix socket instead of stdin/stdout. SSH tunnels to the socket. Reconnection reattaches to same daemon.
- `async-commands`: Non-blocking command execution via `--bg` flag. Job listing, output retrieval, and kill support from CLI.
- `install-prompt`: On tier 2 connect failure, prompt user to install agentd via remote script. Tier 1 remains as fallback if declined.

### Modified Capabilities

## Impact

- **internal/daemon/main.go**: Switch from stdin/stdout protocol loop to Unix socket listener
- **internal/client/session.go**: Connect flow changes — SSH tunnel to socket instead of spawning agentd subprocess
- **internal/client/server.go**: Session server talks to tunneled socket
- **cmd/agentctl/main.go**: New subcommands (jobs, job, kill), --bg flag on run
- **install.sh**: New file — download binary, create systemd user unit, start service
- **Tier 1 unchanged**: bash fallback still works as-is for users who decline install
