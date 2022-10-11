package logx

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io"
	"log"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

var (
	_ Writer = (*mockWriter)(nil)
)

type logEntry struct {
	Level   string      `json:"level"`
	Content interface{} `json:"content"`
}

type mockWriter struct {
	lock    sync.Mutex
	builder strings.Builder
}

func (mw *mockWriter) Error(v interface{}) {
	mw.lock.Lock()
	defer mw.lock.Unlock()
	output(&mw.builder, levelError, v)
}

func (mw *mockWriter) Info(v interface{}) {
	mw.lock.Lock()
	defer mw.lock.Unlock()
	output(&mw.builder, levelInfo, v)
}

func (mw *mockWriter) Close() error {
	return nil
}

func (mw *mockWriter) Contains(text string) bool {
	mw.lock.Lock()
	defer mw.lock.Unlock()
	return strings.Contains(mw.builder.String(), text)
}

func (mw *mockWriter) Reset() {
	mw.lock.Lock()
	defer mw.lock.Unlock()
	mw.builder.Reset()
}

func (mw *mockWriter) String() string {
	mw.lock.Lock()
	defer mw.lock.Unlock()
	return mw.builder.String()
}

func TestFileLineFileMode(t *testing.T) {
	w := new(mockWriter)
	old := writer.Swap(w)
	defer writer.Store(old)

	file, line := getFileLine()
	Error("anything")
	assert.True(t, w.Contains(fmt.Sprintf("%s:%d", file, line+1)))

	file, line = getFileLine()
	Errorf("anything %s", "format")
	assert.True(t, w.Contains(fmt.Sprintf("%s:%d", file, line+1)))
}

func TestFileLineConsoleMode(t *testing.T) {
	w := new(mockWriter)
	old := writer.Swap(w)
	defer writer.Store(old)

	file, line := getFileLine()
	Error("anything")
	assert.True(t, w.Contains(fmt.Sprintf("%s:%d", file, line+1)))

	w.Reset()
	file, line = getFileLine()
	Errorf("anything %s", "format")
	assert.True(t, w.Contains(fmt.Sprintf("%s:%d", file, line+1)))
}

func TestStructedLogError(t *testing.T) {
	w := new(mockWriter)
	old := writer.Swap(w)
	defer writer.Store(old)

	doTestStructedLog(t, levelError, w, func(v ...interface{}) {
		Error(v...)
	})
}

func TestStructedLogErrorf(t *testing.T) {
	w := new(mockWriter)
	old := writer.Swap(w)
	defer writer.Store(old)

	doTestStructedLog(t, levelError, w, func(v ...interface{}) {
		Errorf("%s", fmt.Sprint(v...))
	})
}

func TestStructedLogInfo(t *testing.T) {
	w := new(mockWriter)
	old := writer.Swap(w)
	defer writer.Store(old)

	doTestStructedLog(t, levelInfo, w, func(v ...interface{}) {
		Info(v...)
	})
}

func TestStructedLogInfof(t *testing.T) {
	w := new(mockWriter)
	old := writer.Swap(w)
	defer writer.Store(old)

	doTestStructedLog(t, levelInfo, w, func(v ...interface{}) {
		Infof("%s", fmt.Sprint(v...))
	})
}

func TestStructedLogInfoConsoleText(t *testing.T) {
	w := new(mockWriter)
	old := writer.Swap(w)
	defer writer.Store(old)

	doTestStructedLogConsole(t, w, func(v ...interface{}) {
		old := atomic.LoadUint32(&encoding)
		atomic.StoreUint32(&encoding, plainEncodingType)
		defer func() {
			atomic.StoreUint32(&encoding, old)
		}()

		Info(fmt.Sprint(v...))
	})
}

func TestSetLevel(t *testing.T) {
	SetLevel(ErrorLevel)
	const message = "hello there"
	w := new(mockWriter)
	old := writer.Swap(w)
	defer writer.Store(old)

	Info(message)
	assert.Equal(t, 0, w.builder.Len())
}

func TestSetLevelTwiceWithMode(t *testing.T) {
	testModes := []string{
		"mode",
		"console",
		"volumn",
	}
	w := new(mockWriter)
	old := writer.Swap(w)
	defer writer.Store(old)

	for _, mode := range testModes {
		testSetLevelTwiceWithMode(t, mode, w)
	}
}

func TestSetWriter(t *testing.T) {
	mocked := new(mockWriter)
	SetWriter(mocked)
	assert.Equal(t, mocked, writer.Load())
}

func BenchmarkLogs(b *testing.B) {
	b.ReportAllocs()

	log.SetOutput(io.Discard)
	for i := 0; i < b.N; i++ {
		Info(i)
	}
}

func getFileLine() (string, int) {
	_, file, line, _ := runtime.Caller(1)
	short := file

	for i := len(file) - 1; i > 0; i-- {
		if file[i] == '/' {
			short = file[i+1:]
			break
		}
	}

	return short, line
}

func doTestStructedLog(t *testing.T, level string, w *mockWriter, write func(...interface{})) {
	const message = "hello there"
	write(message)
	var entry logEntry
	if err := json.Unmarshal([]byte(w.String()), &entry); err != nil {
		t.Error(err)
	}
	assert.Equal(t, level, entry.Level)
	val, ok := entry.Content.(string)
	assert.True(t, ok)
	assert.True(t, strings.Contains(val, message))
}

func doTestStructedLogConsole(t *testing.T, w *mockWriter, write func(...interface{})) {
	const message = "hello there"
	write(message)
	assert.True(t, strings.Contains(w.String(), message))
}

func testSetLevelTwiceWithMode(t *testing.T, mode string, w *mockWriter) {
	writer.Store(nil)
	_ = Load(LogConf{
		Mode:  mode,
		Level: "error",
		Path:  "/dev/null",
	})
	_ = Load(LogConf{
		Mode:  mode,
		Level: "info",
		Path:  "/dev/null",
	})
	const message = "hello there"
	Info(message)
	assert.Equal(t, 0, w.builder.Len())
	Infof(message)
	assert.Equal(t, 0, w.builder.Len())
	Error(message)
	assert.Equal(t, 0, w.builder.Len())
	Errorf(message)
	assert.Equal(t, 0, w.builder.Len())
}
