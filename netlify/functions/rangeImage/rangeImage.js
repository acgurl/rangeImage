const fs = require('fs');
const path = require('path');

const handler = async (event) => {
  // 检查请求的来源域名
  const { headers } = event;
  const requestOrigin = headers.origin  || headers.Origin || '';

  // if (requestOrigin !== 'https://www.linguoguang.com')  {
  //   return {
  //     statusCode: 403,
  //     headers: {
  //       'Content-Type': 'application/json',
  //       'Access-Control-Allow-Origin': 'https://www.linguoguang.com', 
  //     },
  //     body: JSON.stringify({  error: 'Access denied.' }),
  //   };
  // }

  try {
    // 读取txt文件中的所有图片链接
    fs.chmod('./ys.txt',  '644', (err) => {
  if (err) {
    console.error(err); 
  } else {
    console.log('File  permissions set successfully.');
    
    const links = fs.readFileSync('./ys.txt',  'utf-8').split('\n').filter(link => link.trim()  !== '');

    // 随机选择一个链接
    const randomIndex = Math.floor(Math.random()  * links.length); 
    const randomLink = links[randomIndex];

    // 302跳转至该链接
    return {
      statusCode: 302,
      headers: {
        'Location': randomLink,
        'Access-Control-Allow-Origin': '*', 
        'Referrer-Policy': no-referrer, 
      },
    };
  } catch (error) {
    console.log(error); 

    // 返回错误消息
    return {
      statusCode: 500,
      headers: {
        'Access-Control-Allow-Methods': 'GET',
        'Access-Control-Allow-Methods': 'POST',
        'Access-Control-Allow-Origin': '*', 
      },
      body: JSON.stringify({ 
        error: 'An error occurred while fetching the random image link.',
      }),
    };
  }
};

module.exports  = { handler };