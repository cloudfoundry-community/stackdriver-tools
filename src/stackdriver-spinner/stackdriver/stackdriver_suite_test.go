package stackdriver_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestStackdriver(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Stackdriver Suite")
}
