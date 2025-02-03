const { MongoClient } = require('mongodb');

// MongoDB 连接配置
const MONGODB_URI = process.env.MONGODB_URI; // 从环境变量获取连接字符串
const DB_NAME = 'api';
const COLLECTION_NAME = 'api_ysh';

// 创建 MongoDB 客户端实例
const client = new MongoClient(MONGODB_URI);

// 随机选择图片URL的函数
const getRandomImageUrl = async () => {
  try {
    await client.connect();
    const collection = client.db(DB_NAME).collection(COLLECTION_NAME);
    
    // 使用聚合框架随机采样一个文档
    const pipeline = [{ $sample: { size: 1 } }];
    const result = await collection.aggregate(pipeline).next();
    
    return result ? result.url : null; // 假设字段名为 url
  } finally {
    await client.close();
  }
};

// 边缘函数处理程序
exports.handler = async (event, context, callback) => {
  try {
    const imageUrl = await getRandomImageUrl();
    
    if (!imageUrl) {
      throw new Error('No image URLs found in database');
    }

    const response = {
      statusCode: 301,
      headers: {
        Location: imageUrl,
      },
    };

    callback(null, response);
  } catch (error) {
    console.error('Error:', error);
    callback(null, {
      statusCode: 500,
      body: JSON.stringify({ error: 'Internal Server Error' })
    });
  }
};