const { MongoClient } = require('mongodb');

const DB_NAME = 'api';
const COLLECTION_NAME = 'api_ysh';

// 添加环境变量验证
const validateEnv = () => {
  if (!process.env.MONGODB_URI) {
    throw new Error('MONGODB_URI is not defined in environment variables');
  }
  if (!process.env.MONGODB_URI.startsWith('mongodb+srv://')) {
    throw new Error('Invalid MongoDB connection string format');
  }
};

const getRandomImageUrl = async () => {
  validateEnv(); // 执行验证
  
  const client = new MongoClient(process.env.MONGODB_URI);
  try {
    await client.connect();
    const collection = client.db(DB_NAME).collection(COLLECTION_NAME);
    const pipeline = [{ $sample: { size: 1 } }];
    const result = await collection.aggregate(pipeline).next();
    return result?.url; // 使用可选链操作符
  } finally {
    await client.close();
  }
};

exports.handler = async (event, context) => {
  try {
    const imageUrl = await getRandomImageUrl();
    if (!imageUrl) {
      return { statusCode: 404, body: 'No image found' };
    }
    return {
      statusCode: 301,
      headers: { Location: imageUrl }
    };
  } catch (error) {
    console.error('Fatal error:', error);
    return {
      statusCode: 500,
      body: JSON.stringify({
        error: error.message,
        stack: process.env.NODE_ENV === 'development' ? error.stack : undefined
      })
    };
  }
};