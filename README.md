# agentd

Persistent remote shells for AI agents. One connection, stateful commands, no SSH gymnastics.

## The problem

Every time an AI agent needs to do something on a remote server:

```bash
ssh -i ~/.ssh/key user@host "cd /app && ls"
ssh -i ~/.ssh/key user@host "cd /app && cat config.py"
ssh -i ~/.ssh/key user@host "cd /app && sudo systemctl restart app"
```

New connection every time. State lost every time. The full SSH command repeated every time.

## The fix

```bash
agentctl connect myserver
agentctl run "cd /app"
agentctl run "ls"                    # still in /app
agentctl run "cat config.py"         # still in /app
agentctl run "sudo systemctl restart app"
agentctl disconnect
```

One connection. State persists. `cd` works. Env vars stick.

## Install

```bash
go install github.com/MrPrinceRawat/agentd/cmd/agentctl@latest
```

## Setup

```bash
# Add a host
agentctl hosts add myserver ubuntu@10.0.0.1 -i ~/.ssh/id_rsa

# Connect (runs in foreground — open a new tab or use &)
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

## Multiple servers

```bash
agentctl connect web-server &
agentctl connect db-server &

agentctl run --on web-server "nginx -t"
agentctl run --on db-server "pg_isready"

agentctl status
# web-server → tier 1 ubuntu@10.0.0.1
# db-server  → tier 1 postgres@10.0.0.2
```

## All commands

```
agentctl connect <name>              Connect to a host
agentctl run <command>               Run on active session
agentctl run --on <name> <command>   Run on specific host
agentctl disconnect [name]           Close session
agentctl hosts add <n> <user@host> [-i key]
agentctl hosts list                  Show configured hosts
agentctl hosts remove <name>         Remove a host
agentctl status                      Show active sessions
```

## Claude Code plugin

Install as a Claude Code plugin:

```bash
claude plugin add MrPrinceRawat/agentd
```

Then just tell Claude: "check the logs on my server" or "restart the app" — it uses `agentctl` automatically. No more raw SSH commands.

Or add the skill manually — copy `skill/SKILL.md` to `.claude/skills/remote/SKILL.md` in your project.

## How it works

`agentctl connect` opens an SSH connection and starts a persistent bash shell on the remote. It runs a local Unix socket server that accepts commands from other `agentctl` processes. Every `agentctl run` sends the command through the socket to the same bash shell — so `cd`, env vars, and aliases persist between calls.

No daemon on the remote. No ports to open. No extra auth. Just SSH.

## Roadmap

**Today (tier 1):** persistent shell over SSH. Works on any server.

**Coming (tier 2):** install `agentd` on the remote for: native file operations, streaming output, background jobs, machine context, smart errors, and granular permissions.

## License

MIT
