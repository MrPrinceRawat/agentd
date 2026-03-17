## ADDED Requirements

### Requirement: agentd listens on Unix socket
agentd SHALL listen on a Unix socket at `/run/user/<uid>/agentd.sock` (or `~/.agentd/agentd.sock` fallback) with 0700 permissions.

#### Scenario: Socket created on startup
- **WHEN** agentd starts
- **THEN** it creates the Unix socket, removes any stale socket file first, and sets permissions to 0700

#### Scenario: Only owning user can connect
- **WHEN** a different user attempts to connect to the socket
- **THEN** the connection is refused by OS-level file permissions

### Requirement: SSH tunnel to Unix socket
agentctl SHALL establish an SSH tunnel forwarding a local socket to the remote agentd socket.

#### Scenario: Tunnel established on connect
- **WHEN** `agentctl connect myserver` detects agentd is installed
- **THEN** it opens an SSH connection with `-L /tmp/agentctl-myserver.sock:/run/user/<uid>/agentd.sock` and communicates through the local socket end

#### Scenario: Reconnect after SSH drop
- **WHEN** SSH tunnel drops and user runs `agentctl connect myserver` again
- **THEN** a new tunnel is established to the same running agentd instance with all state preserved

### Requirement: agentd accepts multiple concurrent connections
agentd SHALL accept multiple connections on the socket simultaneously, each handled in its own goroutine.

#### Scenario: Parallel commands
- **WHEN** two agentctl processes connect to the same agentd socket
- **THEN** both connections are accepted and commands from each are serialized through the shell mutex
