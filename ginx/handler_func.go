package ginx

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"net/http"
	"strconv"
	"zkit/logger"
)

// 受制于泛型，我们这里只能使用包变量，我深恶痛绝的包变量
var log logger.LoggerV1 = logger.NewNoOpLogger()

// 包变量导致我们这个地方的代码非常垃圾
var vector *prometheus.CounterVec

func InitCounter(opt prometheus.CounterOpts) {
	vector = prometheus.NewCounterVec(opt, []string{"code"})
	prometheus.MustRegister(vector)
}

func SetLogger(l logger.LoggerV1) {
	log = l
}

// WrapClaimsAndReq 如果做成中间件来源出去，那么直接耦合 UserClaims 也是不好的。
func WrapClaimsAndReq[Req any](fn func(*gin.Context, Req, UserClaims) (Result, error)) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req Req
		if err := ctx.Bind(&req); err != nil {
			log.Error("解析请求失败", logger.Error(err))
			return
		}
		// 可以用包变量来配置，还是那句话，因为泛型的限制，这里只能用包变量
		rawVal, ok := ctx.Get("user")
		if !ok {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			log.Error("无法获得 claims",
				logger.String("path", ctx.Request.URL.Path))
			return
		}
		// 注意，这里要求放进去 ctx 的不能是*UserClaims，这是常见的一个错误
		claims, ok := rawVal.(UserClaims)
		if !ok {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			log.Error("无法获得 claims",
				logger.String("path", ctx.Request.URL.Path))
			return
		}
		res, err := fn(ctx, req, claims)
		vector.WithLabelValues(strconv.Itoa(res.Code)).Inc()
		if err != nil {
			log.Error("执行业务逻辑失败",
				logger.Error(err))
		}
		ctx.JSON(http.StatusOK, res)
	}
}

// WrapReq 。
func WrapReq[Req any](fn func(*gin.Context, Req) (Result, error)) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req Req
		if err := ctx.Bind(&req); err != nil {
			log.Error("解析请求失败", logger.Error(err))
			return
		}
		res, err := fn(ctx, req)
		if err != nil {
			log.Error("执行业务逻辑失败",
				logger.Error(err))
		}
		vector.WithLabelValues(strconv.Itoa(res.Code)).Inc()
		ctx.JSON(http.StatusOK, res)
	}
}

func Wrap(fn func(*gin.Context) (Result, error)) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		res, err := fn(ctx)
		if err != nil {
			log.Error("执行业务逻辑失败",
				logger.Error(err))
		}
		vector.WithLabelValues(strconv.Itoa(res.Code)).Inc()
		ctx.JSON(http.StatusOK, res)
	}
}

// WrapClaims 复制粘贴
func WrapClaims(fn func(*gin.Context, UserClaims) (Result, error)) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// 可以用包变量来配置，还是那句话，因为泛型的限制，这里只能用包变量
		rawVal, ok := ctx.Get("user")
		if !ok {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			log.Error("无法获得 claims",
				logger.String("path", ctx.Request.URL.Path))
			return
		}
		// 注意，这里要求放进去 ctx 的不能是*UserClaims，这是常见的一个错误
		claims, ok := rawVal.(UserClaims)
		if !ok {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			log.Error("无法获得 claims",
				logger.String("path", ctx.Request.URL.Path))
			return
		}
		res, err := fn(ctx, claims)
		if err != nil {
			log.Error("执行业务逻辑失败",
				logger.Error(err))
		}
		vector.WithLabelValues(strconv.Itoa(res.Code)).Inc()
		ctx.JSON(http.StatusOK, res)
	}
}
