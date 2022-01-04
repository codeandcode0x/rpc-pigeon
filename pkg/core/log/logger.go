package log

import (
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"rpc-gateway/pkg/plugins/httpserver/util"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	DAY_ROTATION = 1140
)

var Log *zap.SugaredLogger

func init() {
	util.InitConfig()
	encoder := zapcore.NewConsoleEncoder(zapcore.EncoderConfig{
		MessageKey:  "msg",
		LevelKey:    "level",
		EncodeLevel: zapcore.CapitalLevelEncoder,
		TimeKey:     "ts",
		NameKey:     "logger",
		EncodeTime: func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(t.Format("2006-01-02 15:04:05"))
		},
		CallerKey:    "file",
		EncodeCaller: zapcore.ShortCallerEncoder,
		EncodeName:   zapcore.FullNameEncoder,
		EncodeDuration: func(d time.Duration, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendInt64(int64(d) / 1000000)
		},
	})

	// get info、error logger's io.Writer
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = viper.GetString("LOG_LEVEL")
	}
	var levelEnabler zapcore.LevelEnabler
	var logWriter io.Writer

	switch logLevel {
	case "info":
		levelEnabler = zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
			return lvl >= zapcore.InfoLevel
		})
		logWriter = getWriter("./logs/log_info.log")
	case "debug":
		levelEnabler = zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
			return lvl >= zapcore.DebugLevel
		})
		logWriter = getWriter("./logs/log_debug.log")
	case "error":
		levelEnabler = zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
			return lvl >= zapcore.ErrorLevel
		})
		logWriter = getWriter("./logs/log_error.log")
	}

	// create Logger
	core := zapcore.NewTee(
		zapcore.NewCore(
			encoder,
			zapcore.NewMultiWriteSyncer(zapcore.AddSync(logWriter), zapcore.AddSync(os.Stdout)),
			levelEnabler),
	)

	// zstack tracing
	caller := zap.AddCaller()
	// get file number
	development := zap.Development()
	// set init fields
	// field := zap.Fields(zap.String("appName", "go-skeleton"))
	Log = zap.New(core, caller, development).Sugar()
}

func getWriter(filename string) io.Writer {
	var logMaxAgeNum, logRotationTimeNum time.Duration
	logMaxAge := os.Getenv("LOG_MAX_AGE")
	if logMaxAge == "" {
		logMaxAgeNum = (time.Duration)(viper.GetInt64("LOG_MAX_AGE"))
	} else {
		logMaxAgeInt64, _ := strconv.ParseInt(logMaxAge, 10, 64)
		logMaxAgeNum = (time.Duration)(logMaxAgeInt64)
	}

	logRotationTime := os.Getenv("LOG_ROTATION_TIME")
	if logRotationTime == "" {
		logRotationTimeNum = (time.Duration)(viper.GetInt64("LOG_ROTATION_TIME"))
	} else {
		logRotationTimeInt64, _ := strconv.ParseInt(logRotationTime, 10, 64)
		logRotationTimeNum = (time.Duration)(logRotationTimeInt64)
	}

	// set log format
	logFormartStr := "%Y%m%d%H%M"
	if int64(logRotationTimeNum)%DAY_ROTATION == 0 {
		logFormartStr = "%Y%m%d"
	}

	hook, err := rotatelogs.New(
		strings.Replace(filename, ".log", "", -1)+"-"+logFormartStr+".log",
		// rotatelogs.WithLinkName(filename),
		// log max age
		rotatelogs.WithMaxAge(time.Minute*logMaxAgeNum),
		// log rotation time
		rotatelogs.WithRotationTime(time.Minute*logRotationTimeNum),
	)

	if err != nil {
		panic(err)
	}
	return hook
}

// log print method
func Debug(args ...interface{}) {
	Log.Debug(args...)
}

func Debugf(template string, args ...interface{}) {
	Log.Debugf(template, args...)
}

func Info(args ...interface{}) {
	Log.Info(args...)
}

func Infof(template string, args ...interface{}) {
	Log.Infof(template, args...)
}

func Warn(args ...interface{}) {
	Log.Warn(args...)
}

func Warnf(template string, args ...interface{}) {
	Log.Warnf(template, args...)
}

func Error(args ...interface{}) {
	Log.Error(args...)
}

func Errorf(template string, args ...interface{}) {
	Log.Errorf(template, args...)
}

func DPanic(args ...interface{}) {
	Log.DPanic(args...)
}

func DPanicf(template string, args ...interface{}) {
	Log.DPanicf(template, args...)
}

func Panic(args ...interface{}) {
	Log.Panic(args...)
}

func Panicf(template string, args ...interface{}) {
	Log.Panicf(template, args...)
}

func Fatal(args ...interface{}) {
	Log.Fatal(args...)
}

func Fatalf(template string, args ...interface{}) {
	Log.Fatalf(template, args...)
}
