package logconfig

import (
	myLogger "github.com/sirupsen/logrus"
)

// This output format is used in the test (has terminal).
func ConfigDebugLogger() {
	// configure log facility in this test
	myLogger.SetReportCaller(true)
	myLogger.SetLevel(myLogger.DebugLevel)
	myLogger.SetFormatter(&myLogger.TextFormatter{
		ForceColors:            true,
		DisableTimestamp:       true,
		DisableLevelTruncation: true,
		PadLevelText:           true,
	})
}

func ConfigInfoLogger() {
	// configure log facility in this test
	myLogger.SetReportCaller(false)
	myLogger.SetLevel(myLogger.InfoLevel)
	myLogger.SetFormatter(&myLogger.TextFormatter{
		ForceColors:            true,
		DisableTimestamp:       true,
		DisableLevelTruncation: true,
		PadLevelText:           true,
	})
}

// This output format is used in production.
func ConfigProductionLogger() {
	// configure log facility in this test
	myLogger.SetLevel(myLogger.InfoLevel)
}
