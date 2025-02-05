const { MongoClient } = require('mongodb');

// MongoDB连接配置
const MONGODB_URI = process.env.MONGODB_URI;
const DB_NAME = 'api';

// 图片类型与数据库集合映射表
const TABLE_MAP = {
  ysh: 'api_ysh', // 原神横屏
  yss: 'api_yss', // 原神竖屏
  xqh: 'api_xqh', // 星穹横屏
  xqs: 'api_xqs'  // 星穹竖屏
};

// MongoDB连接缓存
let cachedClient = null;

// 数据库连接函数
const connectToDatabase = async () => {
  // 如果已有连接则复用
  if (cachedClient) {
    return cachedClient;
  }
  // 创建新连接
  const client = new MongoClient(MONGODB_URI, {
    maxPoolSize: 10,        // 最大连接池大小
    connectTimeoutMS: 5000, // 连接超时时间
    socketTimeoutMS: 5000,  // 套接字超时时间
  });
  cachedClient = await client.connect();
  return cachedClient;
};

// 清理URL，移除所有查询参数
const cleanImageUrl = (url) => {
  try {
    return url.split('?')[0];  // 简单地移除所有查询参数
  } catch (error) {
    console.error('URL清理失败:', error);
    return url;
  }
};

// 获取随机图片URL
const getRandomImageUrl = async (type) => {
  // 验证图片类型是否有效
  if (!type || !TABLE_MAP[type]) {
    throw new Error(`无效类型参数，支持的类型：${Object.keys(TABLE_MAP).join(', ')}`);
  }

  const client = await connectToDatabase();
  const collection = client.db(DB_NAME).collection(TABLE_MAP[type]);

  // 随机获取一条记录
  const result = await collection.aggregate([
    { $sample: { size: 1 } },        // 随机取样
    { $project: { _id: 0, url: 1 } } // 只返回url字段
  ]).next();

  return result?.url ? cleanImageUrl(result.url) : null;
};

// Netlify Functions处理函数
exports.handler = async (event, context) => {
  try {
    const startTime = Date.now();
    const type = event.queryStringParameters?.type?.toLowerCase();
    const imageUrl = await getRandomImageUrl(type);

    // 处理未找到图片的情况
    if (!imageUrl) {
      return {
        statusCode: 404,
        headers: {
          'Access-Control-Allow-Origin': '*',          // 允许跨域访问
          'Referrer-Policy': 'no-referrer',           // 禁止发送Referer
          'Strict-Transport-Security': 'max-age=31536000', // 强制HTTPS
          'X-Content-Type-Options': 'nosniff',        // 禁止MIME类型嗅探
          'X-Frame-Options': 'DENY'                   // 禁止iframe嵌入
        },
        body: JSON.stringify({ error: '未找到图片' })
      };
    }

    console.log(`请求处理时间: ${Date.now() - startTime}ms`);

    // 返回重定向响应
    return {
      statusCode: 302,                // 临时重定向，不保留查询参数
      headers: {
        'Cache-Control': 'no-store',  // 禁止缓存
        'Location': imageUrl,         // 重定向地址
        'Access-Control-Allow-Origin': '*',
        'Referrer-Policy': 'no-referrer',
        'Strict-Transport-Security': 'max-age=31536000',
        'X-Content-Type-Options': 'nosniff',
        'X-Frame-Options': 'DENY',
        'Vary': 'Origin, query'       // 缓存变化依据
      }
    };
  } catch (error) {
    console.error('请求处理失败:', error);
    // 错误响应处理
    return {
      statusCode: error.message.includes('无效类型参数') ? 400 : 500,
      headers: {
        'Access-Control-Allow-Origin': '*',
        'Referrer-Policy': 'no-referrer',
        'Strict-Transport-Security': 'max-age=31536000',
        'X-Content-Type-Options': 'nosniff',
        'X-Frame-Options': 'DENY'
      },
      body: JSON.stringify({
        error: error.message,
        valid_types: Object.keys(TABLE_MAP)
      })
    };
  }
};