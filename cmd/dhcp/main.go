// This source file is part of the Packet Guardian project.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/**
 * This application runs the Packet Guardian DHCP server as a separate process.
 * By default, the main PG binary will not run a DHCP server and it may be better
 * in some circumstances to not allow the main binary to run with root privilages
 * as they are needed to bind to DHCP port 69.
 */

package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/lfkeitel/verbose"
	"github.com/usi-lfkeitel/packet-guardian/src/common"
	"github.com/usi-lfkeitel/packet-guardian/src/models"
	"github.com/usi-lfkeitel/pg-dhcp"
)

var (
	configFile string
	dev        bool
	testConfig bool
	verFlag    bool

	version   = ""
	buildTime = ""
	builder   = ""
	goversion = ""
)

func init() {
	flag.StringVar(&configFile, "c", "", "Configuration file")
	flag.BoolVar(&dev, "d", false, "Run in development mode")
	flag.BoolVar(&testConfig, "t", false, "Test DHCP config")
	flag.BoolVar(&verFlag, "version", false, "Display version information")
	flag.BoolVar(&verFlag, "v", verFlag, "Display version information")
}

func main() {
	flag.Parse()

	if verFlag {
		displayVersionInfo()
		return
	}

	if testConfig {
		testDHCPConfig()
		return
	}

	var err error
	e := common.NewEnvironment(common.EnvProd)
	if dev {
		e.Env = common.EnvDev
	}

	if configFile == "" || !common.FileExists(configFile) {
		configFile = common.FindConfigFile()
	}
	if configFile == "" {
		fmt.Println("No configuration file found")
		os.Exit(1)
	}

	e.Config, err = common.NewConfig(configFile)
	if err != nil {
		fmt.Printf("Error loading configuration: %s\n", err.Error())
		os.Exit(1)
	}

	e.Log = common.NewLogger(e.Config, "dhcp")
	common.SystemLogger = e.Log
	e.Log.Debugf("Configuration loaded from %s", configFile)

	if !common.FileExists(e.Config.DHCP.ConfigFile) {
		e.Log.Fatalf("DHCP configuration file not found: %s", e.Config.DHCP.ConfigFile)
	}

	c := e.SubscribeShutdown()
	go func(e *common.Environment) {
		<-c
		e.Log.Notice("Shutting down...")
		time.Sleep(2)
	}(e)

	e.DB, err = common.NewDatabaseAccessor(e)
	if err != nil {
		e.Log.WithField("error", err).Fatal("Error loading database")
	}
	e.Log.WithFields(verbose.Fields{
		"type":    e.Config.Database.Type,
		"address": e.Config.Database.Address,
	}).Debug("Loaded database")

	dhcpConfig, err := dhcp.ParseFile(e.Config.DHCP.ConfigFile)
	if err != nil {
		e.Log.WithField("error", err).Fatal("Error loading DHCP configuration")
	}

	dhcpPkgConfig := &dhcp.ServerConfig{
		LeaseStore:  models.NewLeaseStore(e),
		DeviceStore: models.NewDHCPDeviceStore(e),
		Env:         dhcp.EnvDev,
		Log:         common.NewLogger(e.Config, "dhcp").Logger,
	}

	handler := dhcp.NewDHCPServer(dhcpConfig, dhcpPkgConfig)
	if err := handler.LoadLeases(); err != nil {
		e.Log.WithField("error", err).Fatal("Couldn't load leases")
	}
	e.Log.Fatal(handler.ListenAndServe())
}

func testDHCPConfig() {
	_, err := dhcp.ParseFile(configFile)
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}

	fmt.Println("Configuration looks good")
}

func displayVersionInfo() {
	fmt.Printf(`Packet Guardian - (C) 2016 The Packet Guardian Authors

Component:   DHCP Server
Version:     %s
Built:       %s
Compiled by: %s
Go version:  %s
`, version, buildTime, builder, goversion)
}
