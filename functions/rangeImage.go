package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// 全局变量定义
var tableMap = map[string]string{
	"ysh":  "api_ysh",  // 原神横屏图片集合
	"yss":  "api_yss",  // 原神竖屏图片集合
	"xqh":  "api_xqh",  // 星穹铁道横屏图片集合
	"xqs":  "api_xqs",  // 星穹铁道竖屏图片集合
	"bing": "api_bing", // 必应每日壁纸集合
}

// 定义支持样式转换的图片类型
var styleEnabledTypes = map[string]bool{
	"ysh": true, // 原神横屏支持样式转换
	"yss": true, // 原神竖屏支持样式转换
	"xqh": true, // 星穹铁道横屏支持样式转换
	"xqs": true, // 星穹铁道竖屏支持样式转换
}

// API错误响应结构
type ApiError struct {
	Code    int    `json:"code"`    // 错误码
	Message string `json:"message"` // 错误信息
	Type    string `json:"type"`    // 错误类型
}

// API统一响应结构
type ApiResponse struct {
	Success bool        `json:"success"`         // 请求是否成功
	Data    interface{} `json:"data,omitempty"`  // 响应数据
	Error   *ApiError   `json:"error,omitempty"` // 错误信息
}

// 系统配置常量
const (
	cacheSize       = 20               // 缓存图片数量
	cacheExpiration = 30 * time.Minute // 缓存过期时间
	maxConnections  = 10               // 最大并发连接数
	rateLimit       = 100              // 每分钟最大请求次数
	cleanupInterval = 5 * time.Minute  // 缓存清理间隔
	maxRetries      = 3                // 最大重试次数
	retryInterval   = time.Second      // 重试等待时间
)

// 缓存项结构
type CacheItem struct {
	URLs      []string  // 缓存的图片URL列表
	UpdatedAt time.Time // 最后更新时间
}

// 缓存管理器
type Cache struct { // 修正：添加 struct 关键字
	sync.RWMutex
	items     map[string]*CacheItem
	lastClean time.Time
}

// 性能指标收集器
type Metrics struct { // 修正：添加 struct 关键字
	sync.RWMutex
	RequestCount    map[string]int
	ResponseTime    map[string]time.Duration
	ErrorCount      map[string]int
	CacheMissCount  map[string]int
	LastCleanup     time.Time
	CacheSize       map[string]int
	AvgResponseTime map[string]time.Duration
}

// 健康状态结构
type HealthStatus struct { // 修正：添加 struct 关键字
	Status     string         `json:"status"`
	CacheStats map[string]int `json:"cache_stats"`
	DBStatus   string         `json:"db_status"`
	Uptime     string         `json:"uptime"`
}

var (
	mongoClient *mongo.Client
	dbName      = "api"
	metrics     = &Metrics{
		RequestCount:   make(map[string]int),
		ResponseTime:   make(map[string]time.Duration),
		ErrorCount:     make(map[string]int),
		CacheMissCount: make(map[string]int),
	}
	// 限流器
	rateLimiter = make(map[string][]time.Time)
	cache       = Cache{items: make(map[string]*CacheItem)}
	// 添加并发控制
	semaphore = make(chan struct{}, maxConnections)
	startTime = time.Now()
)

// 限流检查
func checkRateLimit(clientIP string) bool {
	now := time.Now()
	minute := now.Truncate(time.Minute)

	// 清理过期记录
	var recent []time.Time
	for _, t := range rateLimiter[clientIP] {
		if t.After(minute) {
			recent = append(recent, t)
		}
	}
	rateLimiter[clientIP] = recent

	// 检查限制
	if len(recent) >= rateLimit {
		return false
	}

	rateLimiter[clientIP] = append(rateLimiter[clientIP], now)
	return true
}

// 优化数据库连接
func connectToMongoDB() (*mongo.Client, error) {
	if mongoClient != nil {
		return mongoClient, nil
	}

	// 增加连接池配置
	clientOptions := options.Client().
		ApplyURI(os.Getenv("MONGODB_URI")).
		SetMaxPoolSize(uint64(maxConnections)).
		SetMinPoolSize(2).
		SetMaxConnIdleTime(time.Minute).
		SetRetryWrites(true).
		SetRetryReads(true).
		SetServerSelectionTimeout(5 * time.Second)

		// 使用更长的上下文超时
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("连接MongoDB失败: %v", err)
	}

	// 测试连接
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("MongoDB ping失败: %v", err)
	}

	mongoClient = client
	return client, nil
}

// 批量获取URLs并缓存
func preloadURLs(client *mongo.Client, imageType string) error {
	// 使用更长的上下文超时
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := client.Database(dbName).Collection(tableMap[imageType])

	// 修正：修改 pipeline 语法
	pipeline := mongo.Pipeline{
		{{Key: "$sample", Value: bson.D{{Key: "size", Value: cacheSize}}}},
		{{Key: "$project", Value: bson.D{{Key: "_id", Value: 0}, {Key: "url", Value: 1}}}},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	var urls []string
	for cursor.Next(ctx) {
		var result struct {
			URL string `bson:"url"`
		}
		if err := cursor.Decode(&result); err != nil {
			continue
		}
		urls = append(urls, result.URL)
	}

	if len(urls) > 0 {
		cache.Lock()
		cache.items[imageType] = &CacheItem{
			URLs:      urls,
			UpdatedAt: time.Now(),
		}
		cache.Unlock()
	}

	return nil
}

// 优化缓存获取
func getFromCache(imageType string) (string, bool) {
	cache.RLock()
	defer cache.RUnlock()

	if item, exists := cache.items[imageType]; exists {
		if time.Since(item.UpdatedAt) < cacheExpiration && len(item.URLs) > 0 {
			// 从缓存返回并异步补充
			url := item.URLs[0]
			go func() {
				if len(item.URLs) < cacheSize/2 {
					preloadURLs(mongoClient, imageType)
				}

				cache.Lock()
				item.URLs = item.URLs[1:]
				cache.Unlock()
			}()
			return url, true
		}
	}
	return "", false
}

// 获取随机图片URL优化版
func getRandomImageURL(imageType string) (string, error) {
	if _, ok := tableMap[imageType]; !ok {
		return "", fmt.Errorf("无效类型参数，支持的类型：%v", getValidTypes())
	}

	// 尝试从缓存获取
	if url, hit := getFromCache(imageType); hit {
		return url, nil
	}

	// 缓存未命中，重新加载
	retries := 2
	var lastErr error

	for i := 0; i <= retries; i++ {
		client, err := connectToMongoDB()
		if err != nil {
			lastErr = err
			continue
		}

		err = preloadURLs(client, imageType)
		if err == nil {
			return getRandomImageURL(imageType)
		}
		lastErr = err

		// 短暂等待后重试
		if i < retries {
			time.Sleep(time.Second * time.Duration(i+1))
		}
	}

	return "", fmt.Errorf("获取图片URL失败(重试%d次): %v", retries, lastErr)
}

func getValidTypes() []string {
	types := make([]string, 0, len(tableMap))
	for k := range tableMap {
		types = append(types, k)
	}
	return types
}

// 处理图片URL，插入style参数
func processImageURL(originalURL string, style string, imageType string) string {
	// 如果样式为空或类型不支持样式转换，直接返回原始URL
	if style == "" || !styleEnabledTypes[imageType] {
		return originalURL
	}

	// 解析URL
	u, err := url.Parse(originalURL)
	if err != nil {
		return originalURL
	}

	// 分离基础URL和查询参数
	baseURL := u.Scheme + "://" + u.Host + u.Path
	params := u.Query()

	// 构建新的URL：baseURL + style + 原有参数
	var finalURL string
	if strings.Contains(u.Path, "?") {
		finalURL = baseURL + "@" + style + "&" + params.Encode()
	} else {
		finalURL = baseURL + "@" + style
		if len(params) > 0 {
			finalURL += "?" + params.Encode()
		}
	}

	return finalURL
}

// 优化请求处理函数
func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// 处理健康检查请求
	if request.Path == "/status" {
		return handleHealthCheck(), nil
	}

	// 添加并发控制
	select {
	case semaphore <- struct{}{}:
		defer func() { <-semaphore }()
	case <-time.After(5 * time.Second):
		return errorResponse(503, "服务器繁忙，请稍后重试"), nil
	}

	startTime := time.Now()
	clientIP := request.RequestContext.Identity.SourceIP

	// 清理过期缓存
	cache.cleanup()

	// 限流检查
	if !checkRateLimit(clientIP) {
		return events.APIGatewayProxyResponse{
			StatusCode: 429,
			Body:       `{"error": "Too many requests"}`,
		}, nil
	}

	// 添加请求日志
	fmt.Printf("收到请求: 方法=%s, 路径=%s, 查询参数=%v\n",
		request.HTTPMethod,
		request.Path,
		request.QueryStringParameters,
	)

	// 获取参数
	imageType := strings.ToLower(request.QueryStringParameters["type"])
	style := request.QueryStringParameters["style"]
	wantJSON := request.QueryStringParameters["json"] == "true"

	// 检查style参数
	if style != "" && !styleEnabledTypes[imageType] {
		return errorResponse(400, "此图片类型不支持样式转换"), nil
	}

	// 获取随机图片
	imageURL, err := getRandomImageURL(imageType)
	if err != nil {
		statusCode := 500
		if strings.Contains(err.Error(), "无效类型参数") {
			statusCode = 400
		}

		response := map[string]interface{}{
			"error":       err.Error(),
			"valid_types": getValidTypes(),
		}
		jsonResponse, _ := json.Marshal(response)

		fmt.Printf("请求处理完成: 耗时=%v, 类型=%s, 状态码=%d\n",
			time.Since(startTime),
			imageType,
			statusCode,
		)

		return events.APIGatewayProxyResponse{
			StatusCode: statusCode,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: string(jsonResponse),
		}, nil
	}

	// 处理style参数
	imageURL = processImageURL(imageURL, style, imageType)

	fmt.Printf("请求处理时间: %v, 类型: %s, 样式: %s\n",
		time.Since(startTime),
		imageType,
		style,
	)

	// 记录指标
	defer func() {
		metrics.Lock()
		metrics.RequestCount[imageType]++
		metrics.ResponseTime[imageType] += time.Since(startTime)
		metrics.Unlock()
	}()

	if wantJSON {
		response := map[string]interface{}{
			"code": 200,
			"url":  imageURL,
			"type": imageType,
		}
		jsonResponse, _ := json.Marshal(response)

		return events.APIGatewayProxyResponse{
			StatusCode: 200,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: string(jsonResponse),
		}, nil
	}

	// 默认返回重定向
	return events.APIGatewayProxyResponse{
		StatusCode: 303,
		Headers: map[string]string{
			"Location":                    imageURL,
			"Referrer-Policy":             "no-referrer",
			"Access-Control-Allow-Origin": "*",
		},
	}, nil
}

// 添加清理方法
func (c *Cache) cleanup() {
	if time.Since(c.lastClean) < cleanupInterval {
		return
	}

	c.Lock()
	defer c.Unlock()

	for key, item := range c.items {
		if time.Since(item.UpdatedAt) > cacheExpiration {
			delete(c.items, key)
		}
	}
	c.lastClean = time.Now()
}

// 添加指标导出
func (m *Metrics) export() map[string]interface{} {
	m.RLock()
	defer m.RUnlock()

	return map[string]interface{}{
		"requests":          m.RequestCount,
		"errors":            m.ErrorCount,
		"cache_misses":      m.CacheMissCount,
		"avg_response_time": m.AvgResponseTime,
		"cache_size":        m.CacheSize,
	}
}

// 添加重试机制
func withRetry(operation func() error) error {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		if err := operation(); err == nil {
			return nil
		} else {
			lastErr = err
			time.Sleep(retryInterval * time.Duration(i+1))
		}
	}
	return fmt.Errorf("operation failed after %d retries: %v", maxRetries, lastErr)
}

// 优化数据库连接池
func initMongoDB() error {
	return withRetry(func() error {
		client, err := connectToMongoDB()
		if err != nil {
			return err
		}

		// 尝试创建索引，但不阻止服务启动
		go func() {
			ctx := context.Background()
			for _, collection := range tableMap {
				_, err = client.Database(dbName).Collection(collection).Indexes().CreateOne(ctx,
					mongo.IndexModel{
						Keys:    bson.D{{"url", 1}},
						Options: options.Index().SetUnique(true),
					})
				if err != nil {
					fmt.Printf("警告：创建索引失败（%s）: %v\n", collection, err)
				}
			}
		}()

		// 只要能连接数据库就返回成功
		return nil
	})
}

// 添加安全检查
func isValidRequest(request events.APIGatewayProxyRequest) bool {
	// 检查请求头
	if request.HTTPMethod != "GET" {
		return false
	}

	// 检查参数
	imageType := request.QueryStringParameters["type"]
	if imageType == "" {
		return false
	}

	return true
}

// 统一错误响应
func errorResponse(code int, message string) events.APIGatewayProxyResponse {
	response := ApiResponse{
		Success: false,
		Error: &ApiError{
			Code:    code,
			Message: message,
			Type:    "error",
		},
	}

	jsonResponse, _ := json.Marshal(response)
	return events.APIGatewayProxyResponse{
		StatusCode: code,
		Headers: map[string]string{
			"Content-Type":  "application/json",
			"Cache-Control": "no-store",
		},
		Body: string(jsonResponse),
	}
}

// 添加健康检查处理
func handleHealthCheck() events.APIGatewayProxyResponse {
	status := HealthStatus{
		Status:     "healthy",
		CacheStats: make(map[string]int),
		DBStatus:   "connected",
		Uptime:     time.Since(startTime).String(),
	}

	// 获取缓存统计
	cache.RLock()
	for k, v := range cache.items {
		status.CacheStats[k] = len(v.URLs)
	}
	cache.RUnlock()

	// 检查数据库连接
	if err := mongoClient.Ping(context.Background(), nil); err != nil {
		status.Status = "degraded"
		status.DBStatus = "error: " + err.Error()
	}

	jsonResponse, _ := json.Marshal(status)
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: string(jsonResponse),
	}
}

// 优化错误处理
func withRecovery(handler func(events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error)) func(events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return func(request events.APIGatewayProxyRequest) (response events.APIGatewayProxyResponse, err error) {
		defer func() {
			if r := recover(); r != nil {
				response = errorResponse(500, fmt.Sprintf("内部服务器错误: %v", r))
				err = nil
			}
		}()
		return handler(request)
	}
}

func main() {
	// 初始化数据库连接
	if err := initMongoDB(); err != nil {
		fmt.Printf("数据库连接失败: %v\n", err)
		os.Exit(1)
	}

	// 启动指标收集
	go func() {
		for {
			time.Sleep(time.Minute)
			metrics.export()
		}
	}()

	// 使用恢复机制包装handler
	lambda.Start(withRecovery(handler))
}
