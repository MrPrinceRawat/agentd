## Context

agentd currently runs as a subprocess of SSH — `ssh user@host agentd` spawns the daemon, which reads/writes stdin/stdout. When SSH dies, agentd dies. The persistent shell, jobs, and all state are lost. Commands are synchronous — the agent blocks until completion.

Current architecture: agentctl → Unix socket → session server → SSH pipe → agentd (stdin/stdout)

## Goals / Non-Goals

**Goals:**
- agentd survives SSH disconnects — jobs keep running, shell state preserved
- Non-blocking commands via `--bg` with job management
- Secure by default — Unix socket with file permissions, no open ports
- Simple install — one curl command, prompted on first connect
- Tier 1 (bash fallback) unchanged

**Non-Goals:**
- TCP/port-based transport (Unix socket only — simpler, more secure)
- Multi-user support (single user per daemon)
- Windows support
- agentd auto-update mechanism

## Decisions

### 1. Unix socket over TCP

**Choice:** agentd listens on `/run/user/<uid>/agentd.sock` (or `~/.agentd/agentd.sock` as fallback).

**Why over TCP port:** No port conflicts, no firewall rules, OS-level auth via file permissions (0700). Only the owning user can connect. SSH can tunnel Unix sockets natively: `ssh -L /tmp/agentd-remote.sock:/run/user/1000/agentd.sock user@host`.

### 2. systemd user service over nohup/screen

**Choice:** Install as `~/.config/systemd/user/agentd.service` with `loginctl enable-linger`.

**Why over nohup:** Proper lifecycle management — auto-restart on crash, auto-start on boot, `systemctl --user status agentd` for diagnostics. `enable-linger` keeps user services running even when no SSH session exists.

**Fallback:** If systemd not available (older distros, containers), fall back to nohup + PID file.

### 3. Install via GitHub script, prompted on connect

**Choice:** On `agentctl connect`, if agentd not detected, prompt: "agentd not installed. Install? [y/n]". If yes, run `curl -sSL https://raw.githubusercontent.com/MrPrinceRawat/agentd/main/install.sh | bash` on the remote via SSH.

**Why prompt:** User stays in control. No surprise installations. If declined, falls back to tier 1.

### 4. SSH tunnel as transport layer

**Choice:** `agentctl connect` establishes SSH with `-L` flag to tunnel local socket to remote agentd socket. All protocol messages flow through the tunnel. agentctl talks to the local end of the tunnel.

**Connect flow:**
1. SSH to host
2. Check if agentd socket exists: `test -S /run/user/$(id -u)/agentd.sock`
3. If yes → establish tunnel → tier 2
4. If no → prompt install → if declined → tier 1 bash

### 5. Async commands reuse existing JobManager

**Choice:** `agentctl run --bg "cmd"` sends MsgJOB instead of MsgRUN. Returns job ID. Existing `jobs.go` JobManager already handles background execution, output capture, and kill.

**New CLI surface:**
- `agentctl run --bg "cmd"` → sends MsgJOB, prints job ID
- `agentctl jobs` → sends MsgJOBS, prints table
- `agentctl job <id>` → sends MsgJOBOUT, prints output
- `agentctl kill <id>` → sends MsgKILL

## Risks / Trade-offs

- **systemd not available** → Mitigation: fallback to nohup + PID file management
- **SSH tunnel adds latency** → Negligible for interactive use, ~1ms local socket overhead
- **Stale sockets after crash** → Mitigation: agentd checks and cleans stale socket on startup
- **install.sh running as remote user** → Mitigation: script is public on GitHub, auditable, installs to user-level only (no sudo needed for user service)
- **Shell state after daemon restart** → New bash shell on restart, cwd/env reset. Jobs that were running are lost if daemon itself crashes (acceptable — daemon crash is rare vs SSH drop)
