package xgin

import (
	"github.com/gin-gonic/gin"
	"github.com/smallfish-root/common-pkg/xerror"
	http_logger "github.com/smallfish-root/common-pkg/xgin/http-logger"
	"github.com/smallfish-root/common-pkg/xlogger"
)

func ErrLogger(logger xlogger.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Next()
		context := ctx.Request.Context()
		for _, err := range ctx.Errors {
			ce, ok := err.Err.(*xerror.CustomError)
			if !ok {
				logger.Log(context, xlogger.ErrorLevel, map[string]interface{}{"meta": err.Meta, "error": err.Err}, "系统异常")
			} else {
				fields := map[string]interface{}{
					"surplus": ce.Surplus,
					"meta":    ce.Metadata,
					"code":    ce.Code,
					"error":   ce.GetError(),
				}
				logger.Log(context, xlogger.ErrorLevel, fields, ce.Message)
			}
		}
	}
}

func Logger(logger xlogger.Logger, excludePaths ...string) gin.HandlerFunc {
	l := http_logger.AccessLoggerConfig{
		Logger:         logger,
		BodyLogPolicy:  http_logger.LogAllBodies,
		MaxBodyLogSize: 1024 * 16, //16k
		DropSize:       1024 * 10, //10k
	}

	l.ExcludePaths = map[string]struct{}{}
	for _, excludePath := range excludePaths {
		l.ExcludePaths[excludePath] = struct{}{}
	}
	return http_logger.New(l)
}
