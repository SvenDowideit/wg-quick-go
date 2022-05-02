package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/go-logr/zapr"
	wgquick "github.com/svendowideit/wg-quick-go"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func printHelp() {
	fmt.Print("wg-quick [flags] [ up | down | sync ] [ config_file | interface ]\n\n")
	flag.Usage()
	os.Exit(1)
}

func main() {
	flag.String("iface", "", "interface")
	verbose := flag.Bool("v", false, "verbose")
	protocol := flag.Int("route-protocol", 0, "route protocol to use for our routes")
	metric := flag.Int("route-metric", 0, "route metric to use for our routes")
	flag.Parse()
	args := flag.Args()
	if len(args) != 2 {
		printHelp()
	}

	// setup Logging (chose this to setup for https://pkg.go.dev/go.opentelemetry.io/otel#SetLogger)
	logLevel := 0
	if *verbose {
		logLevel = 2
	}
	zc := zap.NewProductionConfig()
	zc.Level = zap.NewAtomicLevelAt(zapcore.Level(logLevel))
	z, err := zc.Build()
	if err != nil {
		panic(fmt.Sprintf("Failed to initialise logging: %v\n", err))
	}
	log := zapr.NewLogger(z)

	iface := flag.Lookup("iface").Value.String()
	log = log.WithValues("iface", iface)

	cfg := args[1]

	_, err = os.Stat(cfg)
	switch {
	case err == nil:
	case os.IsNotExist(err):
		if iface == "" {
			iface = cfg
			log = log.WithValues("iface", iface)
		}
		cfg = "/etc/wireguard/" + cfg + ".conf"
		_, err = os.Stat(cfg)
		if err != nil {
			log.Error(err, "cannot find config file")
			printHelp()
		}
	default:
		log.Error(err, "error while reading config file")
		printHelp()
	}

	b, err := ioutil.ReadFile(cfg)
	if err != nil {
		log.Error(err, "cannot read file")
	}
	c := &wgquick.Config{}
	if err := c.UnmarshalText(b); err != nil {
		log.Error(err, "cannot parse config file")
	}

	c.RouteProtocol = *protocol
	c.RouteMetric = *metric

	switch args[0] {
	case "up":
		if err := wgquick.Up(c, iface, log); err != nil {
			log.Error(err, "cannot up interface")
		}
	case "down":
		if err := wgquick.Down(c, iface, log); err != nil {
			log.Error(err, "cannot down interface")
		}
	case "sync":
		if err := wgquick.Sync(c, iface, log); err != nil {
			log.Error(err, "cannot sync interface")
		}
	default:
		printHelp()
	}
}
