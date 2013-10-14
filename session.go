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

type OutputWrapper func(io.Reader) io.Reader

func Start(cmd *exec.Cmd) (*Session, error) {
	return StartWrapped(cmd, noopWrapper, noopWrapper)
}

func StartWrapped(cmd *exec.Cmd, outWrapper OutputWrapper, errWrapper OutputWrapper) (*Session, error) {
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

	outExpector := NewExpector(outWrapper(stdout), 0)
	errExpector := NewExpector(errWrapper(stderr), 0)

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

func (s Session) FullOutput() []byte {
	return s.stdout.FullOutput()
}

func (s Session) FullErrorOutput() []byte {
	return s.stderr.FullOutput()
}

func noopWrapper(out io.Reader) io.Reader {
	return out
}
