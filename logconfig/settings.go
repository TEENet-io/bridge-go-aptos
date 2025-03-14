package logconfig

import (
	"fmt"
	"path/filepath"
	"runtime"

	myLogger "github.com/sirupsen/logrus"
)

// This output format is used in the test (has terminal).
func ConfigDebugLogger() {
	// configure log facility in this test
	myLogger.SetReportCaller(true)
	myLogger.SetLevel(myLogger.DebugLevel)
	myLogger.SetFormatter(&myLogger.TextFormatter{
		ForceColors:            true,
		DisableTimestamp:       false,
		FullTimestamp:          true,
		TimestampFormat:        "2006-01-02 15:04:05",
		DisableLevelTruncation: false,
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			// Extract only the file name from the full path
			_, file := filepath.Split(f.File)
			// Extract only the func name from the full path
			funcName := filepath.Base(f.Function)
			return fmt.Sprintf("%s()", funcName), fmt.Sprintf("%s:%d", file, f.Line)
		},
	})
}

func ConfigInfoLogger() {
	// configure log facility in this test
	myLogger.SetReportCaller(false)
	myLogger.SetLevel(myLogger.InfoLevel)
	myLogger.SetFormatter(&myLogger.TextFormatter{
		ForceColors:            true,
		DisableTimestamp:       false,
		FullTimestamp:          true,
		TimestampFormat:        "2006-01-02 15:04:05",
		DisableLevelTruncation: false,
		// PadLevelText:           true,
	})
}

// This output format is used in production.
func ConfigProductionLogger() {
	// configure log facility in this test
	myLogger.SetLevel(myLogger.InfoLevel)
}
