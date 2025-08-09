package logging

import (
	"log"
	"runtime"
	"strings"
)

const (
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorGray   = "\033[90m"
	ColorReset  = "\033[0m"
	LOGERROR    = ColorRed + "ERROR: " + ColorReset
	LOGWARN     = ColorYellow + "WARN: " + ColorReset
	LOGINFO     = ColorBlue + "INFO: " + ColorReset
)

// Logs (s) to the terminal with (arg) arguments, before (s) you have "INFO: "
// printed in Blue
func LogInfo(s string, arg any) {
	logType(LOGINFO, s, arg)
}

// Logs (s) to the terminal with (arg) arguments, before (s) you have "ERROR: "
// printed in Red
func LogError(s string, arg any) {
	logType(LOGERROR, s, arg)
}

// Logs (s) to the terminal with (arg) arguments, before (s) you have "WARN: "
// printed in Yellow
func LogWarn(s string, arg any) {
	logType(LOGWARN, s, arg)
}

// detect which function called the log and log it
func logType(t, s string, arg any) {
	pc, _, _, ok := runtime.Caller(2)
	if !ok {
		log.Printf("%s[%sUnknown%s] %s: %v", t, ColorGray, ColorReset, s, arg)
		return
	}

	fn := runtime.FuncForPC(pc)
	funcName := fn.Name()
	parts := strings.Split(funcName, ".")
	shortName := parts[len(parts)-1]

	log.Printf("%s[%s] %s: %v", t, shortName, s, arg)
}
