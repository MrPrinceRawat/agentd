## 1. Daemon Socket Listener

- [x] 1.1 Add Unix socket listener to agentd — listen on `/run/user/<uid>/agentd.sock` with `~/.agentd/agentd.sock` fallback, 0700 permissions, stale socket cleanup on start
- [x] 1.2 Accept multiple connections on the socket, each in its own goroutine, route to existing protocol handler
- [x] 1.3 Keep existing stdin/stdout mode as fallback (for testing / pipe usage)

## 2. SSH Tunnel Transport

- [x] 2.1 Modify `agentctl connect` to detect agentd socket on remote: `ssh user@host test -S /run/user/$(id -u)/agentd.sock`
- [x] 2.2 If detected, establish SSH tunnel: `-L /tmp/agentctl-<name>.sock:<remote-socket-path>`
- [x] 2.3 Modify session to talk to local tunnel socket instead of SSH stdin/stdout pipes
- [x] 2.4 On reconnect, establish new tunnel to same running daemon — verify state preserved

## 3. Install Prompt & Script

- [x] 3.1 Create `install.sh` — detect arch, download binary from GitHub releases to `~/.agentd/bin/agentd`, create systemd user service, enable-linger, start
- [x] 3.2 Add nohup fallback in install.sh when systemd unavailable
- [x] 3.3 Add install prompt to `agentctl connect` — if socket not found, ask "Install agentd? [y/n]", if yes run install script on remote via SSH
- [x] 3.4 After install, reconnect as tier 2

## 4. Async Commands (CLI)

- [x] 4.1 Add `--bg` flag to `agentctl run` — send MsgJOB instead of MsgRUN, print job ID, return immediately
- [x] 4.2 Add `agentctl jobs` subcommand — send MsgJOBS, display table (id, command, status, duration)
- [x] 4.3 Add `agentctl job <id>` subcommand — send MsgJOBOUT, display output
- [x] 4.4 Add `agentctl kill <id>` subcommand — send MsgKILL, confirm

## 5. Testing & Polish

- [x] 5.1 Test: connect to server with agentd installed, run commands, verify tier 3 works over socket tunnel
- [x] 5.2 Test: kill SSH, reconnect, verify daemon alive and jobs still running
- [x] 5.3 Test: `--bg` flag, job listing, output retrieval
- [ ] 5.4 Test: connect without agentd, accept install prompt, verify install + tier 3
- [ ] 5.5 Test: connect without agentd, decline install, verify tier 1 fallback
- [x] 5.6 Update README with new architecture, install docs, async command examples
