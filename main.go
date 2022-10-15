package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"time"

	"github.com/ffip/hiper"
	"github.com/ffip/hiper/config"
	"github.com/ffip/hiper/util"
	"github.com/sirupsen/logrus"
)

// A version string that can be set with
//
//	-ldflags "-X main.Build=SOMEVERSION"
//
// at compile-time.
var Build string

func main() {
	defaultConfigPath := "/etc/hiper/config.yml"
	if runtime.GOOS == "windows" {
		ex, _ := os.Executable()
		defaultConfigPath = filepath.Dir(ex) + "\\config.yml"
	}

	configPath := flag.String("config", defaultConfigPath, "Path to either a file or directory to load configuration from")
	configTest := flag.Bool("test", false, "Test the config and print the end result. Non zero exit indicates a faulty config")
	printVersion := flag.Bool("version", false, "Print version")
	printUsage := flag.Bool("help", false, "Print command line usage")

	flag.Parse()

	if *printVersion {
		fmt.Printf("Version: %s\n", Build)
		os.Exit(0)
	}

	if *printUsage {
		flag.Usage()
		os.Exit(0)
	}

	l := logrus.New()
	l.Out = os.Stdout

	c := config.NewC(l)
	err := c.Load(*configPath)
	if err != nil {
		l.WithError(err).Error("Failed to load config file")
		flag.Usage()
		os.Exit(1)
	}

	ctrl, err := hiper.Main(c, *configTest, Build, l, nil)

	switch v := err.(type) {
	case util.ContextualError:
		v.Log(l)
		os.Exit(1)
	case error:
		l.WithError(err).Error("Failed to start")
		os.Exit(1)
	}

	if *configTest {
		os.Exit(0)
	}

	go func() {
		ctrl.Start()
		ctrl.ShutdownBlock()
		os.Exit(0)
	}()

	enableRE := regexp.MustCompile(`(?m)^enable: false$`)
	go func() {
		for {
			if cfg, _ := os.ReadFile(*configPath); enableRE.Match(cfg) {
				l.Warn("Config enable status to false")
				ctrl.Stop()
				os.Exit(0)
			}
			time.Sleep(3 * time.Second)
		}
	}()

	var input string
	for {
		fmt.Scanln(&input)
		switch input {
		case "quit":
			l.Warn("Console commandc to stop service")
			ctrl.Stop()
			os.Exit(0)
		case "reload":
			l.Info("Console commandc to reloading config")
			c.ReloadConfig()
		}
	}
}
