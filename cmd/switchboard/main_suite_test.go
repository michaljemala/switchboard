package main_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/fraenkel/candiedyaml"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/pivotal-cf-experimental/switchboard/config"
)

func TestSwitchboard(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Switchboard Executable Suite")
}

var switchboardBinPath string
var switchboardPort uint
var switchboardAPIPort uint
var backends []config.Backend
var configFile string
var proxyConfig config.Proxy
var apiConfig config.API
var pidFile string

var _ = BeforeSuite(func() {
	var err error
	switchboardBinPath, err = gexec.Build("github.com/pivotal-cf-experimental/switchboard/cmd/switchboard", "-race")
	Ω(err).ShouldNot(HaveOccurred())

	switchboardPort = uint(39900 + GinkgoParallelNode())
	switchboardAPIPort = uint(39000 + GinkgoParallelNode())

	backend1 := config.Backend{
		Host:            "localhost",
		Port:            uint(45000 + GinkgoParallelNode()),
		HealthcheckPort: uint(45500 + GinkgoParallelNode()),
		Name:            "backend-0",
	}

	backend2 := config.Backend{
		Host:            "localhost",
		Port:            uint(46000 + GinkgoParallelNode()),
		HealthcheckPort: uint(46500 + GinkgoParallelNode()),
		Name:            "backend-1",
	}

	backends = []config.Backend{backend1, backend2}

	tempDir, err := ioutil.TempDir(os.TempDir(), "switchboard")
	Expect(err).NotTo(HaveOccurred())

	pidFile = filepath.Join(tempDir, "switchboard.pid")
	proxyConfig = config.Proxy{
		Backends:                 backends,
		HealthcheckTimeoutMillis: 500,
		Port: switchboardPort,
	}
	apiConfig = config.API{
		Port:     switchboardAPIPort,
		Username: "username",
		Password: "password",
	}
	rootConfig := config.Root{
		Proxy: proxyConfig,
		API:   apiConfig,
	}

	configFile = filepath.Join(tempDir, "proxyConfig.yml")
	fileToWrite, err := os.Create(configFile)
	Ω(err).ShouldNot(HaveOccurred())

	encoder := candiedyaml.NewEncoder(fileToWrite)
	err = encoder.Encode(rootConfig)
	Ω(err).ShouldNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
})
