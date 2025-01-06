import type { Config, Context } from "@netlify/edge-functions";

const images = [
  "https://i0.hdslb.com/bfs/article/5e587e5511baf9c6366213d78a5468f73493083985480649.jpg",
  "https://i0.hdslb.com/bfs/article/2aa2e85c44a13d1095f1c5fc10e7db653493083985480649.jpg",
  "https://i0.hdslb.com/bfs/article/1e1042c7a0e7ace3e4c0d9c8d2dcbf1a3493083985480649.jpg",
  "https://i0.hdslb.com/bfs/article/f3db656873a69fa6ba55e112d76f7e4f3493083985480649.png",
  "https://i0.hdslb.com/bfs/article/faaf19210b1d07f8695884cd19d2da5e3493083985480649.png",
  "https://i0.hdslb.com/bfs/article/766f1c08349fbd61c26a5be90b3d00843493083985480649.jpg",
  "https://i0.hdslb.com/bfs/article/f561696ef41923c6e59f77aaf24c07383493083985480649.jpg",
  "https://i0.hdslb.com/bfs/article/23299951dc1ca3ebc527e4a7c5dad9083493083985480649.png" 
];


export default async (req: Request, context: Context) => {
  const randomImage = images[Math.floor(Math.random() * images.length)];
  return Response.redirect(randomImage, 301);
};


export const config: Config = {
  path: "/random-image"
};