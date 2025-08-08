package logging

import "log"

const (
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorReset  = "\033[0m"
	LOGERROR    = ColorRed + "ERROR: " + ColorReset
	LOGWARN     = ColorYellow + "WARN: " + ColorReset
	LOGINFO     = ColorBlue + "INFO: " + ColorReset
)

// Logs (s) to the terminal with (arg) arguments, before (s) you have "INFO: "
// printed in Blue
func LogInfo(s string, arg any) {
	log.Printf(LOGINFO+s+": %v", arg)
}

// Logs (s) to the terminal with (arg) arguments, before (s) you have "ERROR: "
// printed in Red
func LogError(s string, arg any) {
	log.Printf(LOGERROR+s+": %v", arg)
}

// Logs (s) to the terminal with (arg) arguments, before (s) you have "WARN: "
// printed in Yellow
func LogWarn(s string, arg any) {
	log.Printf(LOGWARN+s+": %v", arg)
}
