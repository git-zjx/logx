package logx

import (
	"errors"
	"github.com/git-zjx/logx/fs"
	"log"
	"os"
	"path"
	"sync"
)

var ErrLogFileClosed = errors.New("error: log file closed")

type (
	// A DefaultLogger is a Logger.
	DefaultLogger struct {
		filename string
		fp       *os.File
		channel  chan []byte
		done     chan struct{}
		// can't use threading.RoutineGroup because of cycle import
		waitGroup sync.WaitGroup
		closeOnce sync.Once
	}
)

const (
	bufferSize      = 100
	defaultDirMode  = 0o755
	defaultFileMode = 0o600
)

// NewLogger returns a DefaultLogger with given filename and rule, etc.
func NewLogger(filename string) (*DefaultLogger, error) {
	l := &DefaultLogger{
		filename: filename,
		channel:  make(chan []byte, bufferSize),
		done:     make(chan struct{}),
	}
	if err := l.init(); err != nil {
		return nil, err
	}

	l.startWorker()
	return l, nil
}

// Close closes l.
func (l *DefaultLogger) Close() error {
	var err error

	l.closeOnce.Do(func() {
		close(l.done)
		l.waitGroup.Wait()

		if err = l.fp.Sync(); err != nil {
			return
		}

		err = l.fp.Close()
	})

	return err
}

func (l *DefaultLogger) Write(data []byte) (int, error) {
	select {
	case l.channel <- data:
		return len(data), nil
	case <-l.done:
		log.Println(string(data))
		return 0, ErrLogFileClosed
	}
}

func (l *DefaultLogger) init() error {

	if _, err := os.Stat(l.filename); err != nil {
		basePath := path.Dir(l.filename)
		if _, err = os.Stat(basePath); err != nil {
			if err = os.MkdirAll(basePath, defaultDirMode); err != nil {
				return err
			}
		}

		if l.fp, err = os.Create(l.filename); err != nil {
			return err
		}
	} else if l.fp, err = os.OpenFile(l.filename, os.O_APPEND|os.O_WRONLY, defaultFileMode); err != nil {
		return err
	}

	fs.CloseOnExec(l.fp)

	return nil
}

func (l *DefaultLogger) startWorker() {
	l.waitGroup.Add(1)

	go func() {
		defer l.waitGroup.Done()

		for {
			select {
			case event := <-l.channel:
				l.write(event)
			case <-l.done:
				return
			}
		}
	}()
}

func (l *DefaultLogger) write(v []byte) {
	if l.fp != nil {
		_, _ = l.fp.Write(v)
	}
}
