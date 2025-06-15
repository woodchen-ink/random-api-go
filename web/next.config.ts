import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  /* config options here */
  output: 'export',
  trailingSlash: false, // 禁用尾斜杠，避免路由问题
  images: {
    unoptimized: true
  },
  // 在生产环境中不需要代理，因为前后端在同一个服务器上
  ...(process.env.NODE_ENV === 'development' && {
    rewrites: async () => {
      return [
        {
          source: "/api/admin/:path*",
          destination: "http://localhost:5003/api/admin/:path*",
        },
        {
          source: "/api/:path*",
          destination: "http://localhost:5003/api/:path*",
        },
        {
          source: "/pic/:path*",
          destination: "http://localhost:5003/pic/:path*",
        },
        {
          source: "/video/:path*",
          destination: "http://localhost:5003/video/:path*",
        }
      ];
    }
  })
};

export default nextConfig;
