package stackdriver_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestStackdriver(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Stackdriver Suite")
}
