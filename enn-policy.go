package main

import (
	"flag"
	"os"
	"fmt"

	"github.com/spf13/pflag"
	"github.com/golang/glog"

	"enn-policy/app/options"
	"enn-policy/app"
)

func main() {

	config := options.NewEnnPolicyConfig()
	config.AddFlags(pflag.CommandLine)
	flag.CommandLine.Parse([]string{})

	pflag.Parse()
	defer glog.Flush()

	if config.GlogToStderr{
		flag.Set("logtostderr", "true")
	}
	if config.GlogV != "" {
		flag.Set("v",config.GlogV)
	}
	if config.GlogDir != "" {
		flag.Set("log_dir",config.GlogDir)
	}

	if config.CleanupConfig{
		app.CleanUpAndExit()
		os.Exit(0)
	}

	if config.Version{
		app.ShowVersion()
		os.Exit(0)
	}

	s, err := app.NewEnnPolicyServerDefault(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "EnnPolicy config error: %v\n", err)
		os.Exit(1)
	}

	if err = s.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "EnnPolicy run error: %v\n", err)
		os.Exit(1)
	}
}