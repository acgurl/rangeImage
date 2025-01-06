// randomImage.js  
const { createHandler } = require('@netlify/functions');
const fs = require('fs');
const path = require('path');
 
// 获取当前工作目录 
const currentDir = process.cwd(); 
 
// 构建image_urls.json 的完整路径 
const imageDirectory = path.join(currentDir,  'image_urls.json'); 
 
try {
  // 尝试读取图片URL列表 
  const imageUrls = JSON.parse(fs.readFileSync(imageDirectory,  'utf8'));
 
  // 创建一个随机选择图片URL的函数 
  const getRandomImageUrl = () => {
    const randomIndex = Math.floor(Math.random()  * imageUrls.length); 
    return imageUrls[randomIndex];
  };
 
  // 边缘函数处理程序 
  const handler = createHandler((event, context) => {
    // 从列表中随机选择一个图片URL 
    const imageUrl = getRandomImageUrl();
 
    // 返回301重定向响应 
    return {
      statusCode: 301,
      headers: {
        Location: imageUrl,
      },
    };
  });
 
  module.exports  = { handler };
} catch (error) {
  console.error('Error  reading image_urls.json:',  error);
  throw error;
}