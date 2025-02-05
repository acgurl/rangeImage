const { MongoClient } = require('mongodb');

// MongoDB连接配置
const MONGODB_URI = process.env.MONGODB_URI || 'your-default-mongodb-uri';
if (!MONGODB_URI) {
  throw new Error('MONGODB_URI is not defined');
}
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
    socketTimeoutMS: 5000   // 套接字超时时间
  });

  try {
    cachedClient = await client.connect();
    return cachedClient;
  } catch (error) {
    console.error('MongoDB连接失败:', error);
    throw new Error('数据库连接失败');
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
    { $sample: { size: 1 } },
    { $project: { _id: 0, url: 1 } }
  ]).next();

  return result?.url || null;
};

// Netlify Functions处理函数
exports.handler = async (event, context) => {
  try {
    const startTime = Date.now();
    const type = event.queryStringParameters?.type?.toLowerCase();
    const imageUrl = await getRandomImageUrl(type);
    const wantJson = event.queryStringParameters?.json === 'true';

    if (!imageUrl) {
      return {
        statusCode: 404,
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({ error: '未找到图片' })
      };
    }

    console.log(`请求处理时间: ${Date.now() - startTime}ms, 类型: ${type}`);

    // 根据查询参数选择响应格式
    if (wantJson) {
      return {
        statusCode: 200,
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
          code: 200,
          url: imageUrl,
          type: type
        })
      };
    }

    // 默认使用303重定向
    return {
      statusCode: 303,
      headers: {
        'Location': imageUrl
      }
    };
  } catch (error) {
    console.error('请求处理失败:', {
      error: error,
      event: event,
      context: context
    });
    // 错误响应处理
    return {
      statusCode: error.message.includes('无效类型参数') ? 400 : 500,
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        error: error.message,
        valid_types: Object.keys(TABLE_MAP)
      })
    };
  }
};