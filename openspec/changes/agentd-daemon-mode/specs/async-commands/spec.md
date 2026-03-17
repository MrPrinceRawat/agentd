## ADDED Requirements

### Requirement: Non-blocking command execution
agentctl SHALL support `--bg` flag on `run` command to execute commands in the background and return a job ID immediately.

#### Scenario: Background command
- **WHEN** user runs `agentctl run --bg "python train.py"`
- **THEN** agentctl sends MsgJOB, receives a job ID, prints it, and returns immediately

### Requirement: Job listing
agentctl SHALL support `agentctl jobs` to list all background jobs with their status.

#### Scenario: List jobs
- **WHEN** user runs `agentctl jobs`
- **THEN** agentctl displays a table with job ID, command, status (running/done/failed), and duration

### Requirement: Job output retrieval
agentctl SHALL support `agentctl job <id>` to retrieve the output of a background job.

#### Scenario: Get job output
- **WHEN** user runs `agentctl job 3`
- **THEN** agentctl displays the captured stdout/stderr of that job

### Requirement: Job kill
agentctl SHALL support `agentctl kill <id>` to terminate a running background job.

#### Scenario: Kill running job
- **WHEN** user runs `agentctl kill 3` and job 3 is running
- **THEN** the job is terminated and status changes to "killed"
