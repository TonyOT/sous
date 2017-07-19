package sous

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
)

type (
	// ILogger is like this:
	// XXX This is a complete placeholder for work in the ilog branch
	// I needed some extra logging for config process, and didn't want to double
	// down on a process we knew we were going to abandon
	// XXX Further thought: I really think we should look log15 (or something) as our logging platform.
	// It won't be perfect, but it also won't suck up work
	ILogger interface {
		SetLogFunc(func(...interface{}))
		SetDebugFunc(func(...interface{}))
	}

	// LogSet is the stopgap for a decent injectable logger
	LogSet struct {
		Debug  *logwrapper
		Info   *logwrapper
		Warn   *logwrapper
		Notice *logwrapper
		Vomit  *logwrapper

		err   io.Writer
		vomit *log.Logger
		debug *log.Logger
		warn  *log.Logger
	}

	// A temporary type until we can stop using the LogSet loggers directly
	logwrapper struct {
		ffn func(string, ...interface{})
	}
)

var (
	// Log collects various loggers to use for different levels of logging
	// XXX A goal should be to remove this global, and instead inject logging where we need it.
	Log = func() LogSet {
		return *(NewLogSet(os.Stderr))
	}()
)

func (w *logwrapper) Printf(f string, vs ...interface{}) {
	w.ffn(f, vs...)
}

func (w *logwrapper) Print(vs ...interface{}) {
	w.ffn(fmt.Sprint(vs...))
}

func (w *logwrapper) Println(vs ...interface{}) {
	w.ffn(fmt.Sprint(vs...))
}

// SilentLogSet returns a logset that discards everything by default
func SilentLogSet() *LogSet {
	ls := NewLogSet(os.Stderr)
	ls.BeQuiet()
	return ls
}

// NewLogSet builds a new Logset that feeds to the listed writers
func NewLogSet(err io.Writer) *LogSet {
	ls := &LogSet{
		err:   err,
		vomit: log.New(err, "vomit: ", log.Lshortfile|log.Ldate|log.Ltime),
		debug: log.New(err, "debug: ", log.Lshortfile|log.Ldate|log.Ltime),
		warn:  log.New(err, "warn: ", 0),
	}
	ls.Debug = &logwrapper{ffn: ls.debugf}
	ls.Vomit = &logwrapper{ffn: ls.vomitf}
	ls.Warn = &logwrapper{ffn: ls.warnf}
	ls.Info = ls.Warn
	ls.Notice = ls.Warn
	return ls
}

// Vomitf is a simple wrapper on Vomit.Printf
func (ls LogSet) Vomitf(f string, as ...interface{}) { ls.vomit.Output(2, fmt.Sprintf(f, as...)) }
func (ls LogSet) vomitf(f string, as ...interface{}) { ls.vomit.Output(4, fmt.Sprintf(f, as...)) }

// Debugf is a simple wrapper on Debug.Printf
func (ls LogSet) Debugf(f string, as ...interface{}) { ls.debug.Output(2, fmt.Sprintf(f, as...)) }
func (ls LogSet) debugf(f string, as ...interface{}) { ls.debug.Output(4, fmt.Sprintf(f, as...)) }

// Warnf is a simple wrapper on Warn.Printf
func (ls LogSet) Warnf(f string, as ...interface{}) { ls.warn.Output(2, fmt.Sprintf(f, as...)) }
func (ls LogSet) warnf(f string, as ...interface{}) { ls.warn.Output(4, fmt.Sprintf(f, as...)) }

// BeChatty gets the LogSet to print all its output - useful for temporary debugging
func (ls LogSet) BeChatty() {
	ls.warn.SetOutput(ls.err)
	ls.warn.SetFlags(log.Llongfile | log.Ltime)
	ls.vomit.SetOutput(ls.err)
	ls.vomit.SetFlags(log.Llongfile | log.Ltime)
	ls.debug.SetOutput(ls.err)
	ls.debug.SetFlags(log.Llongfile | log.Ltime)
}

// BeHelpful gets the LogSet to print debugging output
func (ls LogSet) BeHelpful() {
	ls.BeQuiet()
	ls.warn.SetOutput(ls.err)
	ls.warn.SetFlags(log.Llongfile | log.Ltime)
	ls.debug.SetOutput(ls.err)
	ls.debug.SetFlags(log.Llongfile | log.Ltime)
}

// BeTerse gets the LogSet to print debugging output
func (ls LogSet) BeTerse() {
	ls.BeQuiet()
	ls.warn.SetOutput(ls.err)
}

// BeQuiet gets the LogSet to discard all its output
func (ls LogSet) BeQuiet() {
	ls.vomit.SetOutput(ioutil.Discard)
	ls.debug.SetOutput(ioutil.Discard)
	ls.warn.SetOutput(ioutil.Discard)
}

// SetupLogging sets up an ILogger to log into the Sous logging regime
func SetupLogging(il ILogger) {
	il.SetLogFunc(func(args ...interface{}) {
		logMaybeMap(Log.Warn, args...)
	})
	il.SetDebugFunc(func(args ...interface{}) {
		logMaybeMap(Log.Debug, args...)
	})
}

func logMaybeMap(l *logwrapper, args ...interface{}) {
	msg, mok := args[0].(string)
	fields, fok := args[1].(map[string]interface{})
	if !(mok && fok) {
		l.Printf(fmt.Sprint(args))
		return
	}
	msg = msg + ": "
	for k, v := range fields {
		msg = fmt.Sprintf("%s %s = %v", msg, k, v)
	}
	l.Printf(msg)
	return
}
