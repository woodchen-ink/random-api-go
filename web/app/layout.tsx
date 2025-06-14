import type { Metadata } from "next";
import "./globals.css";


export const metadata: Metadata = {
  title: "Random-Api 随机API",
  description: "随机图, 随机视频等, 接口丰富快速稳定",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="zh-CN">
      <body
        className={`antialiased`}
      >
        {children}
      </body>
    </html>
  );
}
