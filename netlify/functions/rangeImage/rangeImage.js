// randomImage.js  
const imageUrls = [
  "https://example.com/image1.jpg", 
  "https://example.com/image2.jpg", 
  "https://example.com/image3.jpg" 
];
 
// 创建一个随机选择图片URL的函数 
const getRandomImageUrl = () => {
  const randomIndex = Math.floor(Math.random()  * imageUrls.length); 
  return imageUrls[randomIndex];
};
 
// 边缘函数处理程序 
exports.handler  = (event, context, callback) => {
  // 从列表中随机选择一个图片URL 
  const imageUrl = getRandomImageUrl();
 
  // 返回301重定向响应 
  const response = {
    statusCode: 301,
    headers: {
      Location: imageUrl,
    },
  };
 
  // 使用callback返回响应 
  callback(null, response);
};