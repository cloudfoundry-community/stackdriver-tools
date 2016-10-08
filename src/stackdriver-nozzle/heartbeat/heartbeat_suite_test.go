package heartbeat_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestHeartbeat(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Heartbeat Suite")
}
