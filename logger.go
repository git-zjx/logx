package logx

type Logger interface {
	Error(...interface{})

	Errorf(string, ...interface{})

	Info(...interface{})

	Infof(string, ...interface{})
}
