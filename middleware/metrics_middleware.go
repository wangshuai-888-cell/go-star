package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

var httpRequestDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP 请求耗时分布（秒）",
		Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
	},
	[]string{"method", "path"},
)

var httpRequestsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "HTTP 请求总数，按方法/路由/状态码分组",
	},
	[]string{"method", "path", "status"},
)

func init() {
	prometheus.MustRegister(httpRequestsTotal, httpRequestDuration)
}

// MetricsMiddleware 统计每个请求落在哪个路由、返回什么状态码
func MetricsMiddleware(c *gin.Context) {
	start := time.Now() // 请求进来先记时间
	c.Next()            // 先让请求走完，才能拿到最终状态码

	// 用路由模板而不是真实路径，否则每篇文章 ID 都会生成一条序列，造成基数爆炸
	path := c.FullPath()
	if path == "" {
		path = "unmatched" // 404 等没有匹配路由的请求
	}

	httpRequestsTotal.WithLabelValues(
		c.Request.Method, path, strconv.Itoa(c.Writer.Status()),
	).Inc()

	// 单位必须是秒，指标名也必须以 _seconds 结尾，这是 Prometheus 的约定，Grafana 靠它自动识别单位
	httpRequestDuration.WithLabelValues(
		c.Request.Method, path,
	).Observe(time.Since(start).Seconds())
}
