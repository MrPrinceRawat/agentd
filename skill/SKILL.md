---
name: remote
description: Run commands on remote machines via agentctl. Use when the user wants to check servers, deploy code, read remote logs, run training, or operate on any remote machine.
user-invocable: true
allowed-tools: Bash(agentctl *), Bash(~/Documents/agentd/bin/agentctl *)
---

# Remote Machine Operations

Use `agentctl` (at `~/Documents/agentd/bin/agentctl`) for all remote machine operations. It maintains persistent SSH sessions with state that survives SSH crashes.

## Commands

```bash
agentctl connect <name>              # Connect to a configured host (background with &)
agentctl run "<command>"             # Run on active session (blocking, state persists)
agentctl run --bg "<command>"        # Run in background, returns job ID
agentctl run --on <name> "<command>" # Run on a specific host
agentctl jobs                        # List background jobs
agentctl job <id>                    # Get job output
agentctl kill <id>                   # Kill a background job
agentctl disconnect                  # Close session
agentctl status                      # Show active sessions
agentctl hosts list                  # Show configured hosts
```

## How to use

1. Check configured hosts: `agentctl hosts list`
2. If no session is active, connect: `agentctl connect <name> &`
3. Run commands: `agentctl run "command"` — state persists (cd, env vars)
4. For long tasks: `agentctl run --bg "python train.py"` — returns job ID, doesn't block
5. Check on jobs: `agentctl jobs` and `agentctl job <id>`
6. For multiple servers: `agentctl run --on server1 "cmd"`

## Important

- State persists between `agentctl run` calls — cd works, env vars stick
- Use `--bg` for anything that takes more than a few seconds (training, installs, builds)
- SSH crashes don't kill background jobs — reconnect and check `agentctl jobs`
- Do NOT use raw `ssh` commands — use agentctl instead
- The binary is at `~/Documents/agentd/bin/agentctl`

## Configured hosts

Check with `agentctl hosts list`. Add new hosts with:
```bash
agentctl hosts add <name> <user@host> -i <path-to-key> [-p port]
```
