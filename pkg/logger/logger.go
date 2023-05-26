package logger

import (
	"os"

	"github.com/labstack/gommon/log"
)

// App application logger
var App *log.Logger

// Control channel logger
var Control *log.Logger

func init() {
	lgr := log.New("server")
	lgr.SetHeader(`"level":"${level}","time":"${time_rfc3339_nano}","name":"${prefix}","location":"${short_file}:${line}"}`)
	lgr.SetLevel(getLogLVL())
	App = lgr

	lgr = log.New("control")
	lgr.SetHeader(`"level":"${level}","name":"${prefix}","location":"${short_file}:${line}"}`)
	lgr.SetLevel(getLogLVL())
	Control = lgr
}

func getLogLVL() log.Lvl {
	lvl := os.Getenv("LOG_LEVEL")
	switch lvl {
	case "info":
		return log.INFO
	case "error":
		return log.ERROR
	default:
		return log.DEBUG
	}
}
