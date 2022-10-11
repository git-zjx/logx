package logx

import (
	"errors"
	"fmt"
	"io"
	"runtime/debug"
	"sync"
	"sync/atomic"
)

const (
	InfoLevel uint32 = iota
	ErrorLevel
)

const (
	jsonEncodingType = iota
	plainEncodingType

	plainEncoding = "plain"

	fileMode = "file"
)

type logger struct {
	lw Writer
}

type (
	LogConf struct {
		Mode             string `json:",default=console,options=[console,file]"`
		Encoding         string `json:",default=json,options=[json,plain]"`
		PlainEncodingSep string `json:",default=\t,optional"`
		WithColor        bool   `json:",default=false,optional"`
		TimeFormat       string `json:",optional"`
		Path             string `json:",default=logs"`
		Level            string `json:",default=info,options=[info,error]"`
	}
)

var (
	setupOnce        sync.Once
	logLevel         uint32
	encoding         uint32 = jsonEncodingType
	withColor               = false
	plainEncodingSep        = "\t"
	timeFormat              = "2006-01-02T15:04:05.000Z07:00"
	writer                  = new(atomicWriter)
	conf                    = new(LogConf)
)

// Load 加载日志配置
func Load(c LogConf) (err error) {
	// Just ignore the subsequent SetUp calls.
	// Because multiple services in one process might call SetUp respectively.
	// Need to wait for the first caller to complete the execution.
	setupOnce.Do(func() {
		conf = &c

		setupLogLevel(c)

		setupPath(c)

		setupTimeFormat(c)

		setupPlainEncodingSep(c)

		setupWithColor(c)

		setupEncoding(c)

		err = setupWriter(c)
	})

	return
}

// NewFileLogger 创建新的 file 日志记录
func NewFileLogger(filename string) (*logger, error) {
	if conf == nil {
		return nil, errors.New("config not set")
	}
	w, err := newFileWriter(*conf, filename)
	if err != nil {
		return nil, err
	}
	return &logger{
		lw: w,
	}, nil
}

// Error 记录 Error 级别日志
func (l *logger) Error(v ...interface{}) {
	errorTextSync(l.lw, fmt.Sprint(v...))
}

// Errorf 格式化并记录 Error 级别日志
func (l *logger) Errorf(format string, v ...interface{}) {
	errorTextSync(l.lw, fmt.Errorf(format, v...).Error())
}

// Info 记录 Info 级别日志
func (l *logger) Info(v ...interface{}) {
	infoTextSync(l.lw, fmt.Sprint(v...))
}

// Infof 格式化并记录 Info 级别日志
func (l *logger) Infof(format string, v ...interface{}) {
	infoTextSync(l.lw, fmt.Sprintf(format, v...))
}

// Close 关闭
func (l *logger) Close() error {
	return l.lw.(io.Closer).Close()
}

// Error 记录 Error 级别日志
func Error(v ...interface{}) {
	errorTextSync(getWriter(), fmt.Sprint(v...))
}

// Errorf 格式化并记录 Error 级别日志
func Errorf(format string, v ...interface{}) {
	errorTextSync(getWriter(), fmt.Errorf(format, v...).Error())
}

// Info 记录 Info 级别日志
func Info(v ...interface{}) {
	infoTextSync(getWriter(), fmt.Sprint(v...))
}

// Infof 格式化并记录 Info 级别日志
func Infof(format string, v ...interface{}) {
	infoTextSync(getWriter(), fmt.Sprintf(format, v...))
}

// Close 关闭
func Close() error {
	if w := writer.Swap(nil); w != nil {
		return w.(io.Closer).Close()
	}

	return nil
}

// errorTextSync 写入 Error 级别日志
func errorTextSync(w Writer, msg string) {
	if shallLog(ErrorLevel) {
		w.Error(fmt.Sprintf("%s\n%s", msg, string(debug.Stack())))
	}
}

// infoTextSync 写入 Info 级别日志
func infoTextSync(w Writer, msg string) {
	if shallLog(InfoLevel) {
		w.Info(msg)
	}
}

// getWriter 获取 writer
func getWriter() Writer {
	w := writer.Load()
	if w == nil {
		w = writer.StoreIfNil(newConsoleWriter())
	}
	return w
}

// SetWriter 设置日志 writer，用于自定义日志
func SetWriter(w Writer) {
	writer.Store(w)
}

// SetLevel 设置日志级别
func SetLevel(level uint32) {
	atomic.StoreUint32(&logLevel, level)
}

// shallLog 判断是否可以记录该日志级别
func shallLog(level uint32) bool {
	return atomic.LoadUint32(&logLevel) <= level
}

// setupLogLevel 设置日志级别
func setupLogLevel(c LogConf) {
	switch c.Level {
	case levelInfo:
		SetLevel(InfoLevel)
	case levelError:
		SetLevel(ErrorLevel)
	}
}

func setupPath(c LogConf) {
	if len(c.Path) == 0 {
		c.Path = "logs"
	}
}

func setupTimeFormat(c LogConf) {
	if len(c.TimeFormat) > 0 {
		timeFormat = c.TimeFormat
	}
}

func setupPlainEncodingSep(c LogConf) {
	if len(c.PlainEncodingSep) > 0 {
		plainEncodingSep = c.PlainEncodingSep
	}
}

func setupWithColor(c LogConf) {
	if c.WithColor {
		withColor = c.WithColor
	}
}

func setupEncoding(c LogConf) {
	switch c.Encoding {
	case plainEncoding:
		atomic.StoreUint32(&encoding, plainEncodingType)
	default:
		atomic.StoreUint32(&encoding, jsonEncodingType)
	}
}

func setupWriter(c LogConf) (err error) {
	switch c.Mode {
	case fileMode:
		err = setupWithFiles(c)
	default:
		setupWithConsole()
	}
	return
}

func setupWithConsole() {
	SetWriter(newConsoleWriter())
}

func setupWithFiles(c LogConf) error {
	w, err := newFileWriter(c, "logx")
	if err != nil {
		return err
	}

	SetWriter(w)
	return nil
}
