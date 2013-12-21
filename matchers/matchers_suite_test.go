package cmdtest_matchers_test

import (
	"os/exec"
	"path"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/vito/cmdtest"
)

func TestMatchers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Matchers Suite")
}

func Run(executable string, args ...string) *cmdtest.Session {
	cmd := exec.Command(executable, args...)

	sess, err := cmdtest.Start(cmd)
	if err != nil {
		panic(err)
	}

	return sess
}

func CmdTestSay(name string) *cmdtest.Session {
	return Run(path.Join("assets", name))
}
