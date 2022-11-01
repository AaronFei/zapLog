package zapLog

import (
	"io"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type OptionType_e int
type LogLevel_e int

type LogOption_t struct {
	Option OptionType_e
	Value  interface{}
}

type writerInfo_t struct {
	uid    string
	writer io.Writer
}

const (
	OptionLogLevel OptionType_e = iota
	OptionLogMaxSize
	OptionLogMaxBackup
	OptionLogMaxAge
	OptionLogCompress
	OptionLogDisableSave
	OptionZapOptions
)

const (
	LogLevelDebug LogLevel_e = iota
	LogLevelInfo
)

var optionTable = map[OptionType_e]interface{}{
	OptionLogLevel:       LogLevelInfo,
	OptionLogMaxSize:     1,
	OptionLogMaxBackup:   10,
	OptionLogMaxAge:      30,
	OptionLogCompress:    false,
	OptionLogDisableSave: false,
	OptionZapOptions:     []zap.Option{},
}

var sugarLogger *zap.SugaredLogger
var path string
var zapOptions []zap.Option
var writerList = []writerInfo_t{}

// log level -> -1 = debug, 0 = info
func Init(logPath string, options ...LogOption_t) *zap.SugaredLogger {
	optionHandler(options...)
	path = logPath
	logWriteInit()
	sugarLogger = initLogger(optionTable[OptionZapOptions].([]zap.Option)...)
	return sugarLogger
}

func GetLogger() *zap.SugaredLogger {
	return sugarLogger
}

func ChangeLogLevel(level LogLevel_e) *zap.SugaredLogger {
	sugarLogger.Sync()
	optionTable[OptionLogLevel] = level
	sugarLogger = initLogger(optionTable[OptionZapOptions].([]zap.Option)...)
	return sugarLogger
}

func Close() {
	sugarLogger.Sync()
}

func AddWriter(w io.Writer) (*zap.SugaredLogger, string) {
	uid := uuid.Must(uuid.NewRandom())
	writerList = append(writerList, writerInfo_t{
		uid:    uid.String(),
		writer: w,
	})
	sugarLogger = initLogger(optionTable[OptionZapOptions].([]zap.Option)...)
	return sugarLogger, uid.String()
}

func RemoveWriter(uid string) *zap.SugaredLogger {
	for i, w := range writerList {
		if w.uid == uid {
			writerList = append(writerList[:i], writerList[i+1:]...)
		}
	}
	sugarLogger = initLogger(optionTable[OptionZapOptions].([]zap.Option)...)
	return sugarLogger
}

func initLogger(options ...zap.Option) *zap.SugaredLogger {
	encoder := getEncoder()
	var core zapcore.Core
	if optionTable[OptionLogLevel] == LogLevelDebug {
		core = zapcore.NewCore(encoder, getWriter(), zapcore.DebugLevel)
	} else {
		core = zapcore.NewCore(encoder, getWriter(), zapcore.InfoLevel)
	}

	return zap.New(core, options...).Sugar()
}

func getEncoder() zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = formatEncodeTime
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	return zapcore.NewConsoleEncoder(encoderConfig)
}

func formatEncodeTime(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("2006-01-02 15:04:05"))
}

func logWriteInit() {
	if !optionTable[OptionLogDisableSave].(bool) {
		lumberJackLogger := &lumberjack.Logger{
			Filename:   path,
			MaxSize:    optionTable[OptionLogMaxSize].(int),
			MaxBackups: optionTable[OptionLogMaxBackup].(int),
			MaxAge:     optionTable[OptionLogMaxAge].(int),
			Compress:   optionTable[OptionLogCompress].(bool),
		}
		writerList = append(writerList, writerInfo_t{
			uid:    "",
			writer: lumberJackLogger,
		})
	}
	writerList = append(writerList, writerInfo_t{
		uid:    "",
		writer: os.Stdout,
	})
}

func getWriter() zapcore.WriteSyncer {
	wl := []io.Writer{}
	for _, v := range writerList {
		wl = append(wl, v.writer)
	}
	multiWriter := io.MultiWriter(wl...)

	return zapcore.AddSync(multiWriter)
}

func optionHandler(options ...LogOption_t) {
	for k := range optionTable {
		for _, o := range options {
			if k == o.Option {
				optionTable[k] = o.Value
			}
		}
	}
}
