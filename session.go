package cmdtest

import (
	"fmt"
	"io"
	"os/exec"
	"syscall"
	"time"
)

type Session struct {
	cmd *exec.Cmd

	stdin  io.WriteCloser
	stdout *Expector
	stderr *Expector
}

func Start(executable string, args ...string) (*Session, error) {
	cmd := exec.Command(executable, args...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	outExpector := NewExpector(stdout, 10*time.Second)
	errExpector := NewExpector(stderr, 10*time.Second)

	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	return &Session{
		cmd: cmd,

		stdin:  stdin,
		stdout: outExpector,
		stderr: errExpector,
	}, nil
}

func (s Session) ExpectOutput(pattern string) error {
	return s.stdout.Expect(pattern)
}

func (s Session) ExpectOutputBranches(branches ...ExpectBranch) error {
	return s.stdout.ExpectBranches(branches...)
}

func (s Session) ExpectOutputWithTimeout(pattern string, timeout time.Duration) error {
	return s.stdout.ExpectWithTimeout(pattern, timeout)
}

func (s Session) ExpectError(pattern string) error {
	return s.stderr.Expect(pattern)
}

func (s Session) ExpectErrorWithTimeout(pattern string, timeout time.Duration) error {
	return s.stderr.ExpectWithTimeout(pattern, timeout)
}

func (s Session) Wait(timeout time.Duration) (int, error) {
	exited := make(chan bool)

	go func() {
		s.cmd.Wait()
		exited <- true
	}()

	select {
	case <-exited:
		return s.cmd.ProcessState.Sys().(syscall.WaitStatus).ExitStatus(), nil
	case <-time.After(timeout):
		return -1, fmt.Errorf("command did not exit")
	}
}
