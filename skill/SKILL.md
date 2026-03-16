---
name: remote
description: Run commands on remote machines via agentctl. Use when the user wants to check servers, deploy code, read remote logs, or operate on any remote machine.
user-invocable: true
allowed-tools: Bash(agentctl *)
---

# Remote Machine Operations

Use `agentctl` for all remote machine operations. It maintains persistent SSH sessions with state.

## Commands

```bash
agentctl connect <name>              # Connect to a configured host
agentctl run "<command>"             # Run on active session (state persists — cd works)
agentctl run --on <name> "<command>" # Run on a specific host
agentctl disconnect                  # Close session
agentctl status                      # Show active sessions
```

## How to use

1. If no session is active, connect first: `agentctl connect <name>`
2. Run commands: `agentctl run "command"` — state persists between calls (cd, env vars)
3. For multiple servers: `agentctl run --on server1 "cmd"` and `agentctl run --on server2 "cmd"`

## Important

- State persists between `agentctl run` calls — cd works, env vars stick
- Do NOT use raw `ssh` commands — use agentctl instead
- Check active sessions with `agentctl status` before connecting
