const { MongoClient } = require('mongodb');

const MONGODB_URI = process.env.MONGODB_URI;
const DB_NAME = 'api';

// 表名映射配置
const TABLE_MAP = {
  ysh: 'api_ysh',
  yss: 'api_yss',
  xqh: 'api_xqh',
  xqs: 'api_xqs'
};

// 全局 MongoDB 客户端实例（连接池复用）
let cachedClient = null;

// 初始化 MongoDB 连接
const connectToDatabase = async () => {
  if (cachedClient) {
    return cachedClient;
  }
  const client = new MongoClient(MONGODB_URI, {
    maxPoolSize: 10, // 连接池大小
    connectTimeoutMS: 5000, // 连接超时时间
    socketTimeoutMS: 5000, // 操作超时时间
  });
  cachedClient = await client.connect();
  return cachedClient;
};

// 获取随机图片 URL
const getRandomImageUrl = async (type) => {
  if (!type || !TABLE_MAP[type]) {
    throw new Error(`无效类型参数，支持的类型：${Object.keys(TABLE_MAP).join(', ')}`);
  }

  const client = await connectToDatabase();
  const collection = client.db(DB_NAME).collection(TABLE_MAP[type]);

  // 使用更高效的随机查询方式
  const result = await collection.aggregate([
    { $sample: { size: 1 } },
    { $project: { _id: 0, url: 1 } }
  ]).next();

  return result?.url;
};

// Netlify 函数处理程序
exports.handler = async (event) => {
  try {
    const startTime = Date.now();

    // 从查询参数获取类型
    const type = event.queryStringParameters?.type?.toLowerCase();

    // 获取随机图片 URL
    const imageUrl = await getRandomImageUrl(type);

    if (!imageUrl) {
      return {
        statusCode: 404,
        headers: {
          'Access-Control-Allow-Origin': '*', // 允许跨域
          'Referrer-Policy': 'no-referrer', // 不发送 Referrer 信息
        },
        body: JSON.stringify({ error: '未找到图片' })
      };
    }

    console.log(`请求处理时间: ${Date.now() - startTime}ms`);

    return {
      statusCode: 301,
      headers: {
        'Cache-Control': 'no-cache', // 禁用缓存
        'Location': imageUrl, // 重定向到图片 URL
        'Access-Control-Allow-Origin': '*', // 允许跨域
        'Referrer-Policy': 'no-referrer', // 不发送 Referrer 信息
      }
    };
  } catch (error) {
    console.error('请求处理失败:', error);

    return {
      statusCode: error.message.includes('无效类型参数') ? 400 : 500,
      headers: {
        'Access-Control-Allow-Origin': '*', // 允许跨域
        'Referrer-Policy': 'no-referrer', // 不发送 Referrer 信息
      },
      body: JSON.stringify({
        error: error.message,
        valid_types: Object.keys(TABLE_MAP)
      })
    };
  }
};