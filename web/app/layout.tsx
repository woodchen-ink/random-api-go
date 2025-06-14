import type { Metadata } from "next";
import "./globals.css";


export const metadata: Metadata = {
  title: "Random-Api 随机文件API",
  description: "随机图API, 随机视频等 ",
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
