package ethash

import "github.com/abesuite/abe-miningpool-server/mylog"

// log is a logger that is initialized with no output filters.  This
// means the package will not perform any logging by default until the caller
// requests it.
var log mylog.Logger

// The default amount of logging is none.
func init() {
	DisableLog()
}

// DisableLog disables all library log output.  Logging output is disabled
// by default until UseLogger is called.
func DisableLog() {
	log = mylog.Disabled
}

// UseLogger uses a specified Logger to output package logging info.
func UseLogger(logger mylog.Logger) {
	log = logger
}
