# logx

[go-zero](https://github.com/zeromicro/go-zero) 的 [logx](https://github.com/zeromicro/go-zero/tree/master/core/logx) 精简魔改版

## 安装

```go
go get github.com/git-zjx/logx
```

## 配置说明

```go
LogConf struct {
    Mode             string `json:",default=console,options=[console,file]"`
    Encoding         string `json:",default=json,options=[json,plain]"`
    PlainEncodingSep string `json:",default=\t,optional"`
    WithColor        bool   `json:",default=false,optional"`
    TimeFormat       string `json:",optional"`
    Path             string `json:",default=logs"`
    Level            string `json:",default=info,options=[info,error]"`
}
```

- Mode：输出日志的模式，默认是 console
    - console 模式将日志写到 stdout/stderr
    - file 模式将日志写到 Path 指定目录的文件中
    
- Encoding: 指示如何对日志进行编码，默认是 json
    - json模式以 json 格式写日志
    - plain模式用纯文本写日志，并带有终端颜色显示
    
- WithColor: 指示 plain 模式下是否带终端颜色显示，默认 false
- TimeFormat：自定义时间格式，可选。默认是 2006-01-02T15:04:05.000Z07:00
- Path：设置日志路径，默认为 logs
- Level: 用于过滤日志的日志级别。默认为 info
    - info，所有日志都被写入
    - error, info 的日志被丢弃

## 使用

```go
conf := logx.LogConf {
		Mode:             "file",
		Encoding:         "plain",
		PlainEncodingSep: "\t",
		WithColor:        false,
		Path:             "logs",
	}
	
err := logx.Load(conf)
if err != nil {
    fmt.Println(err)
    return
}

// 写入默认文件，默认为 logx.log
logx.Error("error")

// 写入自定义的文件中
fl, _ := logx.NewFileLogger("test")
fl.Error("error")
```