# agentd

Persistent remote shells for AI agents. One connection, stateful commands, survives SSH crashes.

## The problem

Every time an AI agent needs to do something on a remote server:

```bash
ssh -i ~/.ssh/key user@host "cd /app && ls"
ssh -i ~/.ssh/key user@host "cd /app && cat config.py"
ssh -i ~/.ssh/key user@host "cd /app && sudo systemctl restart app"
```

New connection every time. State lost every time. Long-running commands block everything. SSH drops kill your work.

## The fix

```bash
agentctl connect myserver
agentctl run "cd /app"
agentctl run "ls"                              # still in /app
agentctl run "cat config.py"                   # still in /app
agentctl run --bg "python train.py"            # runs in background, returns job ID
agentctl run "sudo systemctl restart app"      # don't wait for training
agentctl jobs                                  # check on background jobs
agentctl job 1                                 # get training output
```

One connection. State persists. Background jobs survive SSH crashes.

## Install

```bash
go install github.com/MrPrinceRawat/agentd/cmd/agentctl@latest
```

## Setup

```bash
# Add a host
agentctl hosts add myserver ubuntu@10.0.0.1 -i ~/.ssh/id_rsa

# Connect (runs in foreground — use & or a separate tab)
agentctl connect myserver &

# Run commands
agentctl run "whoami"
agentctl run "cd /home/ubuntu/app"
agentctl run "pwd"     # → /home/ubuntu/app
agentctl run "export DB=prod"
agentctl run "echo $DB" # → prod

# Disconnect
agentctl disconnect
```

## Background jobs

Long-running commands don't have to block:

```bash
agentctl run --bg "pip install torch"    # → job 1
agentctl run --bg "python train.py"      # → job 2
agentctl run "echo still working"        # not blocked

agentctl jobs                            # list all jobs
# 1 done  "pip install torch"
# 2 running "python train.py"

agentctl job 1                           # get job output
agentctl kill 2                          # stop a job
```

Requires agentd daemon on the remote (tier 3). On first connect, you'll be prompted to install it.

## SSH crash recovery

With the agentd daemon installed, your work survives SSH disconnects:

```bash
agentctl connect myserver &
agentctl run --bg "python train.py"      # → job 1

# SSH drops, WiFi dies, laptop sleeps — doesn't matter

agentctl connect myserver &              # reconnect
agentctl jobs                            # job still running
# 1 running "python train.py"
agentctl job 1                           # get output so far
```

The daemon runs as a systemd service on the remote. SSH is just a tunnel — when it drops, the daemon keeps going.

## Multiple servers

```bash
agentctl connect web-server &
agentctl connect db-server &

agentctl run --on web-server "nginx -t"
agentctl run --on db-server "pg_isready"

agentctl status
# web-server → tier 3 ubuntu@10.0.0.1
# db-server  → tier 1 postgres@10.0.0.2
```

## How it works

Three tiers, automatic negotiation:

| Tier | What | Requires | Survives SSH crash |
|------|------|----------|--------------------|
| **Tier 1** | Persistent bash over SSH | Nothing | No |
| **Tier 2** | agentd protocol over SSH pipe | `agentd` binary in PATH | No |
| **Tier 3** | agentd daemon via SSH tunnel | `agentd` systemd service | **Yes** |

On `agentctl connect`, the client tries tier 3 first. If the daemon isn't installed, it prompts you to install it. If you decline, it falls back to tier 1.

**Tier 3 architecture:**

```
agentctl ─── SSH tunnel ─── agentd (systemd service)
                │                    │
          just a tunnel         persistent daemon
          can drop/reconnect    owns shell + jobs
                                survives everything
```

## Installing agentd on a remote

Automatic (prompted on first connect):
```
$ agentctl connect myserver
agentd not installed on myserver. Install? [y/n] y
Installing agentd on myserver...
Install complete.
Connected to myserver (tier 3)
```

Manual:
```bash
curl -sSL https://raw.githubusercontent.com/MrPrinceRawat/agentd/main/install.sh | bash
```

The install script downloads the binary, creates a systemd user service, and enables it.

## All commands

```
agentctl connect <name>              Connect to a host
agentctl run <command>               Run on active session (blocking)
agentctl run --bg <command>          Run in background (returns job ID)
agentctl run --on <name> <command>   Run on specific host
agentctl jobs                        List background jobs
agentctl job <id>                    Get job output
agentctl kill <id>                   Kill a background job
agentctl disconnect [name]           Close session
agentctl hosts add <n> <user@host> [-i key] [-p port]
agentctl hosts list                  Show configured hosts
agentctl hosts remove <name>         Remove a host
agentctl status                      Show active sessions
```

## Claude Code plugin

```bash
claude plugin add MrPrinceRawat/agentd
```

Then just tell Claude: "check the logs on my server" or "run training in the background" — it uses `agentctl` automatically.

## License

CC BY-NC 4.0 — free for non-commercial use.
