package network

import (
	"io"
	"os/exec"
)

// HostCommandExecutor isolates the host utilities used by the Linux network
// adapter. OEM/platform compositions can supply a compatible implementation
// without changing network policy or the public service entrypoints.
type HostCommandExecutor interface {
	Run(name string, args ...string) error
	Output(name string, args ...string) ([]byte, error)
	RunWithInput(name string, input io.Reader, args ...string) error
}

// OSCommandExecutor is the current host implementation and preserves the
// existing exec.Command behavior.
type OSCommandExecutor struct{}

func (OSCommandExecutor) Run(name string, args ...string) error {
	return exec.Command(name, args...).Run()
}

func (OSCommandExecutor) Output(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).Output()
}

func (OSCommandExecutor) RunWithInput(name string, input io.Reader, args ...string) error {
	command := exec.Command(name, args...)
	command.Stdin = input
	return command.Run()
}

func defaultHostCommandExecutor(executors ...HostCommandExecutor) HostCommandExecutor {
	if len(executors) > 0 && executors[0] != nil {
		return executors[0]
	}
	return OSCommandExecutor{}
}
