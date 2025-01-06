// randomImage.js  
const { createHandler } = require('@netlify/functions');
 
// 硬编码的图片URL列表 
const imageUrls = [
  "https://i0.hdslb.com/bfs/article/5e587e5511baf9c6366213d78a5468f73493083985480649.jpg",
  "https://i0.hdslb.com/bfs/article/2aa2e85c44a13d1095f1c5fc10e7db653493083985480649.jpg",
  "https://i0.hdslb.com/bfs/article/1e1042c7a0e7ace3e4c0d9c8d2dcbf1a3493083985480649.jpg",
  "https://i0.hdslb.com/bfs/article/f3db656873a69fa6ba55e112d76f7e4f3493083985480649.png",
  "https://i0.hdslb.com/bfs/article/faaf19210b1d07f8695884cd19d2da5e3493083985480649.png",
  "https://i0.hdslb.com/bfs/article/766f1c08349fbd61c26a5be90b3d00843493083985480649.jpg",
  "https://i0.hdslb.com/bfs/article/f561696ef41923c6e59f77aaf24c07383493083985480649.jpg",
  "https://i0.hdslb.com/bfs/article/23299951dc1ca3ebc527e4a7c5dad9083493083985480649.png" 
];
 
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