## ADDED Requirements

### Requirement: Prompt to install on first connect
agentctl SHALL prompt the user to install agentd when connecting to a host where agentd is not detected.

#### Scenario: agentd not installed, user accepts
- **WHEN** `agentctl connect myserver` fails to detect agentd socket on the remote
- **THEN** agentctl prompts "agentd not installed on myserver. Install? [y/n]"
- **WHEN** user responds "y"
- **THEN** agentctl runs the install script on the remote via SSH, then establishes tier 2 connection

#### Scenario: agentd not installed, user declines
- **WHEN** user responds "n" to the install prompt
- **THEN** agentctl falls back to tier 1 (bash) connection

### Requirement: Install script on GitHub
The project SHALL include an `install.sh` script that downloads the correct agentd binary, installs it, and sets up the systemd user service.

#### Scenario: Install on Linux amd64
- **WHEN** `install.sh` runs on a Linux amd64 system
- **THEN** it downloads the amd64 binary to `~/.agentd/bin/agentd`, creates `~/.config/systemd/user/agentd.service`, runs `systemctl --user enable --now agentd`, and runs `loginctl enable-linger`

#### Scenario: Install on Linux arm64
- **WHEN** `install.sh` runs on a Linux arm64 system
- **THEN** it downloads the arm64 binary and follows the same setup
