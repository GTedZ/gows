package websockets

import (
	"fmt"
	"os"
	"time"

	"golang.org/x/sys/windows"
)

func init() {
	stdout := windows.Handle(os.Stdout.Fd())
	var originalMode uint32

	windows.GetConsoleMode(stdout, &originalMode)
	windows.SetConsoleMode(stdout, originalMode|0x0004)
}

const (
	nothing_LVL = iota
	shouldnt_happen_LVL
	error_LVL
	warn_LVL
	important_LVL
	info_LVL
	debug_LVL
)

var Logger = GoLogger{
	// PrintLogsLevel: debug_LVL,
	// LogLevel:       important_LVL,
}

type GoLogger struct {
	disabled bool

	PrintLogsLevel int

	LogLevel int
	LogFile  string
}

func (logger *GoLogger) DEBUG(message string, err ...error) {
	logger.log(debug_LVL, message, err...)
}

func (logger *GoLogger) INFO(message string, err ...error) {
	logger.log(info_LVL, message, err...)
}

func (logger *GoLogger) IMPORTANT(message string, err ...error) {
	logger.log(important_LVL, message, err...)
}

func (logger *GoLogger) WARN(message string, err ...error) {
	logger.log(warn_LVL, message, err...)
}

func (logger *GoLogger) ERROR(message string, err ...error) {
	logger.log(error_LVL, message, err...)
}

func (logger *GoLogger) SHOULDNT_HAPPEN(message string, err ...error) {
	logger.log(shouldnt_happen_LVL, message, err...)
}

func (logger *GoLogger) format_DDMMYYYY_HHMMSSMS(time time.Time) string {
	time = time.UTC()

	return fmt.Sprintf("%02d/%02d/%d %02d:%02d:%02d.%03d", time.Day(), int(time.Month()), time.Year(), time.Hour(), time.Minute(), time.Second(), time.UnixMilli()%1000)
}

func (logger *GoLogger) Enable() {
	logger.disabled = false
}

func (logger *GoLogger) Disable() {
	logger.disabled = true
}

func (logger *GoLogger) log(level int, message string, errs ...error) {
	if logger.disabled {
		return
	}

	var log_lvl_str = ""
	switch level {
	case debug_LVL:
		log_lvl_str = "DEBUG"
	case info_LVL:
		log_lvl_str = "INFO"
	case important_LVL:
		log_lvl_str = "IMPORTANT"
	case warn_LVL:
		log_lvl_str = "WARN"
	case error_LVL:
		log_lvl_str = "ERROR"
	case shouldnt_happen_LVL:
		log_lvl_str = "SHOULDNT_HAPPEN"
	}

	str := fmt.Sprintf("[%s] %s: %s\n", log_lvl_str, logger.format_DDMMYYYY_HHMMSSMS(time.Now()), message)
	for i, err := range errs {
		errIndex_str := ""
		if len(errs) > 1 {
			errIndex_str = fmt.Sprintf(" %d", i+1)
		}
		str += fmt.Sprintf("\t\t\t\t=> err%s: %s\n", errIndex_str, err.Error())
	}

	var Color string
	switch level {
	case debug_LVL:
		Color = "\x1b[90m"
	case info_LVL:
		Color = "\x1b[37m"
	case important_LVL:
		Color = "\x1b[32m"
	case warn_LVL:
		Color = "\x1b[33m"
	case error_LVL:
		Color = "\x1b[31m"
	case shouldnt_happen_LVL:
		Color = "\x1b[31m"
	}

	if logger.PrintLogsLevel >= level {
		fmt.Println(Color + str + "\x1b[0m")
	}

	if logger.LogLevel >= level {
		logger.appendLogFile(str)
	}
}

func (logger *GoLogger) appendLogFile(str string) {
	fileName := "logs.txt"
	if logger.LogFile != "" {
		fileName = logger.LogFile
	}

	file, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("[GoLogger] Error opening file:", err)
		return
	}
	defer file.Close()

	// Write data to file
	_, err = file.WriteString(str)
	if err != nil {
		fmt.Println("[GoLogger] Error writing to file:", err)
		return
	}
}
