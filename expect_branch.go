package cmdtest

import (
	"regexp"
	"time"
)

type ExpectBranch struct {
	Pattern string
	Callback func()

	stop chan bool

	pattern *regexp.Regexp
}

func (b *ExpectBranch) match(result chan func(), expector *Expector) {
	matched := b.matchOutput(expector)

	if matched {
		result <- b.Callback
	} else {
		result <- nil
	}
}

func (b *ExpectBranch) cancel() {
	select {
	case b.stop <- true:
	default:
	}
}

func (b *ExpectBranch) matchOutput(expector *Expector) bool {
	for {
		found := b.pattern.FindIndex(expector.nextOutput())
		if found != nil {
			expector.forwardOutput(found[1])
			return true
		}

		if expector.closed {
			return false
		}

		select {
		case <-time.After(100 * time.Millisecond):
		case <-b.stop:
			return false
		}
	}
}
