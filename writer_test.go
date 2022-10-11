package logx

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/stretchr/testify/assert"
	"log"
	"testing"
)

func TestNewWriter(t *testing.T) {
	const literal = "foo bar"
	var buf bytes.Buffer
	w := NewWriter(&buf)
	w.Info(literal)
	assert.Contains(t, buf.String(), literal)
}

func TestConsoleWriter(t *testing.T) {
	var buf bytes.Buffer
	w := newConsoleWriter()
	lw := newLogWriter(log.New(&buf, "", 0))
	w.(*defaultWriter).lw = lw
	w.Error("foo bar 1")
	var val mockedEntry
	if err := json.Unmarshal(buf.Bytes(), &val); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, levelError, val.Level)
	assert.Equal(t, "foo bar 1", val.Content)

	buf.Reset()
	w.(*defaultWriter).lw = lw
	w.Info("foo bar 2")
	if err := json.Unmarshal(buf.Bytes(), &val); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, levelInfo, val.Level)
	assert.Equal(t, "foo bar 2", val.Content)

	w.(*defaultWriter).lw = hardToCloseWriter{}
	assert.NotNil(t, w.Close())
	w.(*defaultWriter).lw = easyToCloseWriter{}
}

func TestWriteJson(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	writeJson(nil, "foo")
	assert.Contains(t, buf.String(), "foo")
	buf.Reset()
	writeJson(nil, make(chan int))
	assert.Contains(t, buf.String(), "unsupported type")
}

func TestWritePlainAny(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	writePlainAny(nil, levelInfo, "foo")
	assert.Contains(t, buf.String(), "foo")

	buf.Reset()
	writePlainAny(nil, levelError, make(chan int))
	assert.Contains(t, buf.String(), "unsupported type")
}

type mockedEntry struct {
	Level   string `json:"level"`
	Content string `json:"content"`
}

type easyToCloseWriter struct{}

func (h easyToCloseWriter) Write(_ []byte) (_ int, _ error) {
	return
}

func (h easyToCloseWriter) Close() error {
	return nil
}

type hardToCloseWriter struct{}

func (h hardToCloseWriter) Write(_ []byte) (_ int, _ error) {
	return
}

func (h hardToCloseWriter) Close() error {
	return errors.New("close error")
}

type hardToWriteWriter struct{}

func (h hardToWriteWriter) Write(_ []byte) (_ int, _ error) {
	return 0, errors.New("write error")
}
