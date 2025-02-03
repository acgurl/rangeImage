const { MongoClient } = require('mongodb');
const { URL } = require('url'); // 新增 URL 解析模块

const MONGODB_URI = process.env.MONGODB_URI;
const DB_NAME = 'api';

// 表名映射配置
const TABLE_MAP = {
  ysh: 'api_ysh',
  yss: 'api_yss',
  xqh: 'api_xqh',
  xqs: 'api_xqs'
};

let cachedClient = null;

const connectToDatabase = async () => {
  if (cachedClient) {
    return cachedClient;
  }
  const client = new MongoClient(MONGODB_URI, {
    maxPoolSize: 10,
    connectTimeoutMS: 5000,
    socketTimeoutMS: 5000,
  });
  cachedClient = await client.connect();
  return cachedClient;
};

// 新增 URL 清理函数
const sanitizeImageUrl = (url) => {
  try {
    const parsedUrl = new URL(url);
    // 清除所有查询参数
    parsedUrl.search = '';
    return parsedUrl.toString();
  } catch (error) {
    console.error('Invalid image URL:', url);
    throw new Error('图片链接格式错误');
  }
};

const getRandomImageUrl = async (type) => {
  if (!type || !TABLE_MAP[type]) {
    throw new Error(`无效类型参数，支持的类型：${Object.keys(TABLE_MAP).join(', ')}`);
  }

  const client = await connectToDatabase();
  const collection = client.db(DB_NAME).collection(TABLE_MAP[type]);

  const result = await collection.aggregate([
    { $sample: { size: 1 } },
    { $project: { _id: 0, url: 1 } }
  ]).next();

  // 清理图片 URL
  return result?.url ? sanitizeImageUrl(result.url) : null;
};

exports.handler = async (event) => {
  try {
    const startTime = Date.now();
    const type = event.queryStringParameters?.type?.toLowerCase();
    const imageUrl = await getRandomImageUrl(type);

    if (!imageUrl) {
      return {
        statusCode: 404,
        headers: {
          'Access-Control-Allow-Origin': '*',
          'Referrer-Policy': 'no-referrer'
        },
        body: JSON.stringify({ error: '未找到图片' })
      };
    }

    console.log(`请求处理时间: ${Date.now() - startTime}ms`);

    return {
      statusCode: 301,
      headers: {
        'Cache-Control': 'no-cache',
        'Location': imageUrl,
        'Access-Control-Allow-Origin': '*',
        'Referrer-Policy': 'no-referrer'
      }
    };
  } catch (error) {
    console.error('请求处理失败:', error);
    return {
      statusCode: error.message.includes('无效类型参数') ? 400 : 500,
      headers: {
        'Access-Control-Allow-Origin': '*',
        'Referrer-Policy': 'no-referrer'
      },
      body: JSON.stringify({
        error: error.message,
        valid_types: Object.keys(TABLE_MAP)
      })
    };
  }
};