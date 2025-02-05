package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
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
)

// 数据库连接函数
func connectToMongoDB() (*mongo.Client, error) {
	if mongoClient != nil {
		return mongoClient, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client().
		ApplyURI(os.Getenv("MONGODB_URI")).
		SetMaxPoolSize(10).
		SetConnectTimeout(5 * time.Second).
		SetSocketTimeout(5 * time.Second)

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, err
	}

	// 测试连接
	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, err
	}

	mongoClient = client
	return client, nil
}

// 获取随机图片URL
func getRandomImageURL(imageType string) (string, error) {
	collectionName, ok := tableMap[imageType]
	if !ok {
		return "", fmt.Errorf("无效类型参数，支持的类型：%v", getValidTypes())
	}

	client, err := connectToMongoDB()
	if err != nil {
		return "", err
	}

	collection := client.Database(dbName).Collection(collectionName)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 聚合查询随机获取一条记录
	pipeline := mongo.Pipeline{
		{{"$sample", bson.D{{"size", 1}}}},
		{{"$project", bson.D{{"_id", 0}, {"url", 1}}}},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return "", err
	}
	defer cursor.Close(ctx)

	var result struct {
		URL string `bson:"url"`
	}

	if cursor.Next(ctx) {
		err = cursor.Decode(&result)
		if err != nil {
			return "", err
		}
		return result.URL, nil
	}

	return "", fmt.Errorf("未找到图片")
}

func getValidTypes() []string {
	types := make([]string, 0, len(tableMap))
	for k := range tableMap {
		types = append(types, k)
	}
	return types
}

// 处理函数
func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	startTime := time.Now()

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
