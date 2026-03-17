## ADDED Requirements

### Requirement: agentd runs as a systemd user service
agentd SHALL run as a persistent systemd user service that survives SSH disconnects and auto-restarts on crash.

#### Scenario: Service persists after SSH disconnect
- **WHEN** agentd is running as a systemd service and the SSH connection drops
- **THEN** agentd continues running, the persistent shell and all background jobs remain alive

#### Scenario: Service auto-restarts on crash
- **WHEN** agentd crashes unexpectedly
- **THEN** systemd restarts it automatically and a new shell is initialized

#### Scenario: Service starts on boot with linger
- **WHEN** the server reboots and `loginctl enable-linger` is configured
- **THEN** agentd starts automatically without requiring an SSH login

### Requirement: Fallback to nohup when systemd unavailable
agentd SHALL fall back to nohup with PID file management when systemd user services are not available.

#### Scenario: No systemd
- **WHEN** the install script detects systemd is not available
- **THEN** it starts agentd via `nohup` and writes a PID file to `~/.agentd/agentd.pid`
