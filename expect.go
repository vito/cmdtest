package cmdtest

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"
	"time"
)

type Expector struct {
	output         io.Reader
	defaultTimeout time.Duration

	closed bool

	offset     int
	buffer     *bytes.Buffer
	fullBuffer *bytes.Buffer

	sync.RWMutex
}

type ExpectationFailed struct {
	Branches []ExpectBranch
	Next   string
	Output string
}

func (e ExpectationFailed) Error() string {
	patterns := []string{}

	for _, branch := range e.Branches {
		patterns = append(patterns, branch.Pattern)
	}

	return fmt.Sprintf(
		"Expected to see '%s', got stuck at: %#v.\n\nFull output:\n\n%s",
		strings.Join(patterns, "' or '"),
		e.Next,
		e.Output,
	)
}

func NewExpector(out io.Reader, defaultTimeout time.Duration) *Expector {
	e := &Expector{
		output:         out,
		defaultTimeout: defaultTimeout,

		buffer:     new(bytes.Buffer),
		fullBuffer: new(bytes.Buffer),
	}

	go e.monitor()

	return e
}

func (e *Expector) Expect(pattern string) error {
	return e.ExpectWithTimeout(pattern, e.defaultTimeout)
}

func (e *Expector) ExpectWithTimeout(pattern string, timeout time.Duration) error {
	return e.ExpectBranchesWithTimeout(
		timeout,
		ExpectBranch{
			Pattern: pattern,
			Callback: func() {},
		},
	)

}

func (e *Expector) ExpectBranches(branches ...ExpectBranch) error {
	return e.ExpectBranchesWithTimeout(e.defaultTimeout, branches...)
}

func (e *Expector) ExpectBranchesWithTimeout(timeout time.Duration, branches ...ExpectBranch) error {
	matchResults := make(chan func())

	for _, branch := range branches {
		re, err := regexp.Compile(branch.Pattern)
		if err != nil {
			return err
		}

		branch.stop = make(chan bool)
		branch.pattern = re

		go branch.match(matchResults, e)
	}

	matchedCallback := make(chan func())
	allComplete := make(chan bool)

	go consumeResults(branches, matchResults, matchedCallback, allComplete)

	select {
	case callback := <-matchedCallback:
		callback()
		return nil
	case <-allComplete:
		return e.failedMatch(branches)
	case <-time.After(timeout):
		for _, b := range branches {
			b.cancel()
		}

		return e.failedMatch(branches)
	}
}

func consumeResults(branches []ExpectBranch, matchResults chan func(), matchedCallback chan func(), allComplete chan bool) {
	sentCallback := false

	for _ = range branches {
		result := <-matchResults

		if result != nil && !sentCallback {
			sentCallback = true
			matchedCallback <- result
		}
	}

	allComplete <- true
}

func (e *Expector) failedMatch(branches []ExpectBranch) ExpectationFailed {
	return ExpectationFailed{
		Branches: branches,
		Next:   string(e.nextOutput()),
		Output: string(e.fullOutput()),
	}
}

func (e *Expector) monitor() {
	var buf [1024]byte

	for {
		read, err := e.output.Read(buf[:])

		if read > 0 {
			e.addOutput(buf[:read])
		}

		if err != nil {
			break
		}
	}

	e.closed = true
}

func (e *Expector) addOutput(out []byte) {
	e.Lock()
	defer e.Unlock()

	e.buffer.Write(out)
	e.fullBuffer.Write(out)
}

func (e *Expector) forwardOutput(count int) {
	e.Lock()
	defer e.Unlock()

	e.buffer.Next(count)
}

func (e *Expector) nextOutput() []byte {
	e.RLock()
	defer e.RUnlock()

	return e.buffer.Bytes()
}

func (e *Expector) fullOutput() []byte {
	e.RLock()
	defer e.RUnlock()

	return e.fullBuffer.Bytes()
}
