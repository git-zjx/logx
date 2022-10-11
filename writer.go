package logx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/git-zjx/logx/color"
	"io"
	"log"
	"path"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	fatihColor "github.com/fatih/color"
)

const (
	callerKey    = "caller"
	callerDepth  = 5
	contentKey   = "content"
	levelKey     = "level"
	timestampKey = "@timestamp"
)

var (
	levelInfo  = "info"
	levelError = "error"

	flags = 0x0
)

type (
	Writer interface {
		Close() error
		Error(v interface{})
		Info(v interface{})
	}

	atomicWriter struct {
		writer Writer
		lock   sync.RWMutex
	}

	defaultWriter struct {
		lw io.WriteCloser
	}
)

func (w *atomicWriter) Load() Writer {
	w.lock.RLock()
	defer w.lock.RUnlock()
	return w.writer
}

func (w *atomicWriter) Store(v Writer) {
	w.lock.Lock()
	defer w.lock.Unlock()
	w.writer = v
}

func (w *atomicWriter) StoreIfNil(v Writer) Writer {
	w.lock.Lock()
	defer w.lock.Unlock()

	if w.writer == nil {
		w.writer = v
	}

	return w.writer
}

func (w *atomicWriter) Swap(v Writer) Writer {
	w.lock.Lock()
	defer w.lock.Unlock()
	old := w.writer
	w.writer = v
	return old
}

func (w *defaultWriter) Close() error {
	return w.lw.Close()
}

func (w *defaultWriter) Error(v interface{}) {
	output(w.lw, levelError, v)
}

func (w *defaultWriter) Info(v interface{}) {
	output(w.lw, levelInfo, v)
}

func NewWriter(w io.Writer) Writer {
	lw := newLogWriter(log.New(w, "", flags))

	return &defaultWriter{
		lw: lw,
	}
}

func newConsoleWriter() Writer {
	lw := newLogWriter(log.New(fatihColor.Output, "", flags))
	return &defaultWriter{
		lw: lw,
	}
}

func newFileWriter(c LogConf, filename string) (Writer, error) {
	var err error
	var lw io.WriteCloser

	if len(c.Path) == 0 {
		c.Path = "logs"
	}

	filePath := path.Join(c.Path, filename) + ".log"

	setupLogLevel(c)

	if lw, err = createOutput(filePath); err != nil {
		return nil, err
	}

	return &defaultWriter{
		lw: lw,
	}, nil
}

func createOutput(path string) (io.WriteCloser, error) {
	return NewLogger(path)
}

func output(writer io.Writer, level string, val interface{}) {

	switch atomic.LoadUint32(&encoding) {
	case plainEncodingType:
		writePlainAny(writer, level, val)
	default:
		entry := make(map[string]interface{})
		entry[timestampKey] = getTimestamp()
		entry[levelKey] = level
		entry[contentKey] = val
		entry[callerKey] = getCaller(callerDepth)
		writeJson(writer, entry)
	}
}

func writePlainAny(writer io.Writer, level string, val interface{}) {
	if withColor {
		level = wrapLevelWithColor(level)
	}

	switch v := val.(type) {
	case string:
		writePlainText(writer, level, v)
	case error:
		writePlainText(writer, level, v.Error())
	case fmt.Stringer:
		writePlainText(writer, level, v.String())
	default:
		writePlainValue(writer, level, v)
	}
}

func writePlainText(writer io.Writer, level, msg string) {
	var buf bytes.Buffer
	buf.WriteString(getTimestamp())
	buf.WriteString(plainEncodingSep)
	buf.WriteString(level)
	buf.WriteString(plainEncodingSep)
	buf.WriteString(msg)
	buf.WriteString(plainEncodingSep)
	buf.WriteString(fmt.Sprintf("%s=%v", callerKey, getCaller(callerDepth)))
	buf.WriteByte('\n')
	if writer == nil {
		log.Println(buf.String())
		return
	}

	if _, err := writer.Write(buf.Bytes()); err != nil {
		log.Println(err.Error())
	}
}

func writePlainValue(writer io.Writer, level string, val interface{}) {
	var buf bytes.Buffer
	buf.WriteString(getTimestamp())
	buf.WriteString(plainEncodingSep)
	buf.WriteString(level)
	buf.WriteString(plainEncodingSep)
	if err := json.NewEncoder(&buf).Encode(val); err != nil {
		log.Println(err.Error())
		return
	}
	buf.WriteString(plainEncodingSep)
	buf.WriteString(fmt.Sprintf("%s=%v", callerKey, getCaller(callerDepth)))
	buf.WriteByte('\n')
	if writer == nil {
		log.Println(buf.String())
		return
	}

	if _, err := writer.Write(buf.Bytes()); err != nil {
		log.Println(err.Error())
	}
}

func wrapLevelWithColor(level string) string {
	var colour color.Color
	switch level {
	case levelError:
		colour = color.FgRed
	case levelInfo:
		colour = color.FgBlue
	}

	if colour == color.NoColor {
		return level
	}

	return color.WithColorPadding(level, colour)
}

func writeJson(writer io.Writer, info interface{}) {
	if content, err := json.Marshal(info); err != nil {
		log.Println(err.Error())
	} else if writer == nil {
		log.Println(string(content))
	} else {
		_, _ = writer.Write(append(content, '\n'))
	}
}

func getCaller(callDepth int) string {
	_, file, line, ok := runtime.Caller(callDepth)
	if !ok {
		return ""
	}

	return prettyCaller(file, line)
}

func getTimestamp() string {
	return time.Now().Format(timeFormat)
}

func prettyCaller(file string, line int) string {
	idx := strings.LastIndexByte(file, '/')
	if idx < 0 {
		return fmt.Sprintf("%s:%d", file, line)
	}

	idx = strings.LastIndexByte(file[:idx], '/')
	if idx < 0 {
		return fmt.Sprintf("%s:%d", file, line)
	}

	return fmt.Sprintf("%s:%d", file[idx+1:], line)
}
