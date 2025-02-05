package main

import (
	"context"
	"encoding/json"
	"fmt"
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client().
		ApplyURI(mongoURI).
		SetMaxPoolSize(5).          // 减小连接池大小
		SetMinPoolSize(1).          // 保持至少一个连接
		SetMaxConnIdleTime(time.Minute). // 空闲连接超时
		SetConnectTimeout(2*time.Second).
		SetSocketTimeout(2*time.Second)

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, err
	}

	mongoClient = client
	return client, nil
}

// 批量获取URLs并缓存
func preloadURLs(client *mongo.Client, imageType string) error {
	collection := client.Database(dbName).Collection(tableMap[imageType])
	
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

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
	client, err := connectToMongoDB()
	if err != nil {
		return "", err
	}

	if err := preloadURLs(client, imageType); err != nil {
		return "", err
	}

	return getRandomImageURL(imageType)
}

func getValidTypes() []string {
	types := make([]string, 0, len(tableMap))
	for k := range tableMap {
		types = append(types, k)
	}
	return types
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

	fmt.Printf("请求处理时间: %v, 类型: %s\n", time.Since(startTime), imageType)

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
		},
	}, nil
}

func main() {
	lambda.Start(handler)
}
