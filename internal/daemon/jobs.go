package daemon

import (
	"bytes"
	"fmt"
	"os/exec"
	"sync"
)

type Job struct {
	ID      int
	Command string
	Status  string // "running", "done", "failed"
	Exit    int
	Output  bytes.Buffer
	cmd     *exec.Cmd
}

type JobManager struct {
	jobs   map[int]*Job
	nextID int
	mu     sync.Mutex
}

func NewJobManager() *JobManager {
	return &JobManager{
		jobs:   make(map[int]*Job),
		nextID: 1,
	}
}

// Start launches a command in the background
func (jm *JobManager) Start(command string) int {
	jm.mu.Lock()
	id := jm.nextID
	jm.nextID++

	job := &Job{
		ID:      id,
		Command: command,
		Status:  "running",
	}

	cmd := exec.Command("bash", "-c", command)
	cmd.Stdout = &job.Output
	cmd.Stderr = &job.Output
	job.cmd = cmd

	jm.jobs[id] = job
	jm.mu.Unlock()

	go func() {
		err := cmd.Run()
		jm.mu.Lock()
		defer jm.mu.Unlock()

		if err != nil {
			job.Status = "failed"
			if exitErr, ok := err.(*exec.ExitError); ok {
				job.Exit = exitErr.ExitCode()
			} else {
				job.Exit = -1
			}
		} else {
			job.Status = "done"
			job.Exit = 0
		}
	}()

	return id
}

// List returns all jobs
func (jm *JobManager) List() []*Job {
	jm.mu.Lock()
	defer jm.mu.Unlock()

	result := make([]*Job, 0, len(jm.jobs))
	for _, j := range jm.jobs {
		result = append(result, j)
	}
	return result
}

// GetOutput returns a job's current output
func (jm *JobManager) GetOutput(id int) (string, error) {
	jm.mu.Lock()
	defer jm.mu.Unlock()

	job, ok := jm.jobs[id]
	if !ok {
		return "", fmt.Errorf("job %d not found", id)
	}
	return job.Output.String(), nil
}

// Kill stops a running job
func (jm *JobManager) Kill(id int) error {
	jm.mu.Lock()
	defer jm.mu.Unlock()

	job, ok := jm.jobs[id]
	if !ok {
		return fmt.Errorf("job %d not found", id)
	}
	if job.cmd.Process != nil {
		return job.cmd.Process.Kill()
	}
	return nil
}
