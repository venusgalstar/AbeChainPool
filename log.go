// Copyright (c) 2021 The Abelian Foundation
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/abesuite/abe-miningpool-server/chainclient"
	"github.com/abesuite/abe-miningpool-server/configmgr"
	"github.com/abesuite/abe-miningpool-server/consensus/ethash"
	"github.com/abesuite/abe-miningpool-server/dal"
	"github.com/abesuite/abe-miningpool-server/minermgr"
	"github.com/abesuite/abe-miningpool-server/model"
	"github.com/abesuite/abe-miningpool-server/mylog"
	"github.com/abesuite/abe-miningpool-server/ordermgr"
	"github.com/abesuite/abe-miningpool-server/poolserver"
	"github.com/abesuite/abe-miningpool-server/rewardmgr"
	"github.com/abesuite/abe-miningpool-server/rpcclient"
	"github.com/abesuite/abe-miningpool-server/service"
	"github.com/abesuite/abe-miningpool-server/utils"
	"github.com/abesuite/abe-miningpool-server/walletclient"

	"github.com/jrick/logrotate/rotator"
)

// logWriter implements an io.Writer that outputs to both standard output and
// the write-end pipe of an initialized log rotator.
type logWriter struct{}

func (logWriter) Write(p []byte) (n int, err error) {
	os.Stdout.Write(p)
	logRotator.Write(p)
	return len(p), nil
}

// Loggers per subsystem.  A single backend logger is created and all subsytem
// loggers created from it will write to the backend.  When adding new
// subsystems, add the subsystem logger variable here and to the
// subsystemLoggers map.
//
// Loggers can not be used before the log rotator has been initialized with a
// log file.  This must be performed early during application startup by calling
// initLogRotator.
var (
	// backendLog is the logging backend used to create all subsystem loggers.
	// The backend must not be used before the log rotator has been initialized,
	// or data races and/or nil pointer dereferences will occur.
	backendLog = mylog.NewBackend(logWriter{})

	// logRotator is one of the logging outputs.  It should be closed on
	// application shutdown.
	logRotator *rotator.Rotator

	poolLog         = backendLog.Logger("POOL")
	chainLog        = backendLog.Logger("CHNS")
	rpcClientLog    = backendLog.Logger("RPCCLIENT")
	walletClientLog = backendLog.Logger("WALLETCLIENT")
	poolServerLog   = backendLog.Logger("POSV")
	minerMgrLog     = backendLog.Logger("MIMG")
	dalLog          = backendLog.Logger("DAL")
	modelLog        = backendLog.Logger("MODEL")
	utilsLog        = backendLog.Logger("UTILS")
	rewardMgrLog    = backendLog.Logger("REWARD")
	serviceLog      = backendLog.Logger("SERVICE")
	dagLog          = backendLog.Logger("DAG")
	orderLog        = backendLog.Logger("ORDER")
	configLog       = backendLog.Logger("CONFIG")
)

// Initialize package-global logger variables.
func init() {
	chainclient.UseLogger(chainLog)
	rpcclient.UseLogger(rpcClientLog)
	walletclient.UseLogger(walletClientLog)
	poolserver.UseLogger(poolServerLog)
	minermgr.UseLogger(minerMgrLog)
	dal.UseLogger(dalLog)
	model.UseLogger(modelLog)
	utils.UseLogger(utilsLog)
	rewardmgr.UseLogger(rewardMgrLog)
	service.UseLogger(serviceLog)
	ethash.UseLogger(dagLog)
	ordermgr.UseLogger(orderLog)
	configmgr.UseLogger(configLog)
}

// subsystemLoggers maps each subsystem identifier to its associated logger.
var subsystemLoggers = map[string]mylog.Logger{
	"POOL":      poolLog,
	"RPCCLIENT": rpcClientLog,
	"CHNS":      chainLog,
	"POSV":      poolServerLog,
	"MIMG":      minerMgrLog,
	"DAL":       dalLog,
	"MODEL":     modelLog,
	"UTILS":     utilsLog,
	"REWARD":    rewardMgrLog,
	"SERVICE":   serviceLog,
	"DAG":       dagLog,
	"ORDER":     orderLog,
	"CONFIG":    configLog,
}

// initLogRotator initializes the logging rotater to write logs to logFile and
// create roll files in the same directory.  It must be called before the
// package-global log rotater variables are used.
func initLogRotator(logFile string) {
	logDir, _ := filepath.Split(logFile)
	err := os.MkdirAll(logDir, 0700)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create log directory: %v\n", err)
		os.Exit(1)
	}
	r, err := rotator.New(logFile, 10*1024, false, 30)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create file rotator: %v\n", err)
		os.Exit(1)
	}

	logRotator = r
}

// setLogLevel sets the logging level for provided subsystem.  Invalid
// subsystems are ignored.  Uninitialized subsystems are dynamically created as
// needed.
func setLogLevel(subsystemID string, logLevel string) {
	// Ignore invalid subsystems.
	logger, ok := subsystemLoggers[subsystemID]
	if !ok {
		return
	}

	// Defaults to info if the log level is invalid.
	level, _ := mylog.LevelFromString(logLevel)
	logger.SetLevel(level)
}

// setLogLevels sets the log level for all subsystem loggers to the passed
// level.  It also dynamically creates the subsystem loggers as needed, so it
// can be used to initialize the logging system.
func setLogLevels(logLevel string) {
	// Configure all sub-systems with the new logging level.  Dynamically
	// create loggers as needed.
	for subsystemID := range subsystemLoggers {
		setLogLevel(subsystemID, logLevel)
	}
}

// directionString is a helper function that returns a string that represents
// the direction of a connection (inbound or outbound).
func directionString(inbound bool) string {
	if inbound {
		return "inbound"
	}
	return "outbound"
}

// pickNoun returns the singular or plural form of a noun depending
// on the count n.
func pickNoun(n uint64, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}
