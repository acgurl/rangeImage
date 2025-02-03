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

// 验证环境和参数
const validateInput = (type) => {
  if (!process.env.MONGODB_URI) {
    throw new Error('MONGODB_URI 未配置');
  }
  
  if (!type || !TABLE_MAP[type]) {
    throw new Error(`无效类型参数，支持的类型：${Object.keys(TABLE_MAP).join(', ')}`);
  }
};

const getRandomImageUrl = async (type) => {
  validateInput(type);
  
  const client = new MongoClient(MONGODB_URI);
  try {
    await client.connect();
    const collection = client.db(DB_NAME).collection(TABLE_MAP[type]);
    
    // 使用更高效的随机查询方式
    const result = await collection.aggregate([
      { $sample: { size: 1 } },
      { $project: { _id: 0, url: 1 } }
    ]).next();

    return result?.url;
  } finally {
    await client.close();
  }
};

exports.handler = async (event) => {
  try {
    // 从查询参数获取类型
    const type = event.queryStringParameters?.type?.toLowerCase();
    
    // 获取随机图片URL
    const imageUrl = await getRandomImageUrl(type);
    
    if (!imageUrl) {
      return {
        statusCode: 404,
        body: JSON.stringify({ error: '未找到图片' })
      };
    }

    return {
      statusCode: 301,
      headers: {
        'Cache-Control': 'no-cache',
        Location: imageUrl
      }
    };
  } catch (error) {
    console.error('请求处理失败:', error);
    
    return {
      statusCode: error.message.includes('无效类型参数') ? 400 : 500,
      body: JSON.stringify({
        error: error.message,
        valid_types: Object.keys(TABLE_MAP)
      })
    };
  }
};