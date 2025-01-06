// 引入所需的模块 
const { createHandler } = require('@netlify/functions');
 
// 定义一个包含图片URLs的数组 
const imageUrls = [
  "https://i0.hdslb.com/bfs/article/5e587e5511baf9c6366213d78a5468f73493083985480649.jpg",
  "https://i0.hdslb.com/bfs/article/2aa2e85c44a13d1095f1c5fc10e7db653493083985480649.jpg",
  "https://i0.hdslb.com/bfs/article/1e1042c7a0e7ace3e4c0d9c8d2dcbf1a3493083985480649.jpg",
  "https://i0.hdslb.com/bfs/article/f3db656873a69fa6ba55e112d76f7e4f3493083985480649.png",
  "https://i0.hdslb.com/bfs/article/faaf19210b1d07f8695884cd19d2da5e3493083985480649.png",
  "https://i0.hdslb.com/bfs/article/766f1c08349fbd61c26a5be90b3d00843493083985480649.jpg",
  "https://i0.hdslb.com/bfs/article/f561696ef41923c6e59f77aaf24c07383493083985480649.jpg",
  "https://i0.hdslb.com/bfs/article/23299951dc1ca3ebc527e4a7c5dad9083493083985480649.png" 
  // 可以添加更多的图片URL 
];
 
// 创建一个边缘函数处理程序 
const handler = createHandler(async (event) => {
  // 从请求中获取URL路径参数 
  const path = event.path; 
  
  // 检查路径是否符合预期，如果不是，则返回404 
  if (path !== '/random-image') {
    return {
      status: 404,
      body: 'Not Found',
    };
  }
  
  // 从imageUrls数组中随机选择一个图片URL 
  const randomImageUrl = imageUrls[Math.floor(Math.random() * imageUrls.length)]; 
  
  // 返回301永久重定向响应到随机选择的图片URL 
  return {
    status: 301,
    headers: {
      'Location': randomImageUrl,
    },
  };
});
 
// 导出handler函数 
module.exports  = { handler };