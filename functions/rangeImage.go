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

// 图片类型与集合映射
var tableMap = map[string]string{
	"ysh": "api_ysh", // 原神横屏
	"yss": "api_yss", // 原神竖屏
	"xqh": "api_xqh", // 星穹横屏
	"xqs": "api_xqs", // 星穹竖屏
}

var (
	mongoClient *mongo.Client
	dbName      = "api"
	// 添加缓存
	cache       = struct {
		sync.RWMutex
		urls map[string][]string
	}{urls: make(map[string][]string)}
	cacheSize   = 10  // 每个类型缓存10个URL
)

// 数据库连接函数优化
func connectToMongoDB() (*mongo.Client, error) {
	if mongoClient != nil {
		return mongoClient, nil
	}

	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" {
		return nil, fmt.Errorf("MONGODB_URI 环境变量未设置")
	}

	clientOptions := options.Client().
		ApplyURI(mongoURI).
		SetMaxPoolSize(5).
		SetMinPoolSize(1).
		SetMaxConnIdleTime(time.Minute).
		SetConnectTimeout(5*time.Second).     // 增加连接超时
		SetServerSelectionTimeout(5*time.Second).  // 增加服务器选择超时
		SetSocketTimeout(10*time.Second)          // 增加套接字超时

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
	
	pipeline := mongo.Pipeline{
		{{"$sample", bson.D{{"size", cacheSize}}}},
		{{"$project", bson.D{{"_id", 0}, {"url", 1}}}},
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
		cache.urls[imageType] = urls
		cache.Unlock()
	}

	return nil
}

// 获取随机图片URL优化版
func getRandomImageURL(imageType string) (string, error) {
	if _, ok := tableMap[imageType]; !ok {
		return "", fmt.Errorf("无效类型参数，支持的类型：%v", getValidTypes())
	}

	// 尝试从缓存获取
	cache.RLock()
	urls, exists := cache.urls[imageType]
	cache.RUnlock()

	if exists && len(urls) > 0 {
		// 从缓存随机返回一个URL
		url := urls[0]
		
		// 异步更新缓存
		go func() {
			cache.Lock()
			cache.urls[imageType] = urls[1:]
			cache.Unlock()

			// 如果缓存不足，异步补充
			if len(urls) < cacheSize/2 {
				if client, err := connectToMongoDB(); err == nil {
					preloadURLs(client, imageType)
				}
			}
		}()

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
func processImageURL(originalURL string, style string) string {
    if style == "" {
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

// 处理函数优化
func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	startTime := time.Now()
	
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
    imageURL = processImageURL(imageURL, style)

	fmt.Printf("请求处理时间: %v, 类型: %s, 样式: %s\n", 
		time.Since(startTime), 
		imageType,
		style,
	)

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
			"Location": imageURL,
			"Referrer-Policy": "no-referrer",
			"Access-Control-Allow-Origin": "*",
		},
	}, nil
}

func main() {
	lambda.Start(handler)
}
