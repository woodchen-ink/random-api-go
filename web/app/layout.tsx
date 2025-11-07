import type { Metadata } from "next";
import "./globals.css";
import Script from "next/script";


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
      <Script async src="https://l.czl.net/script.js" data-website-id="fd458fd1-2228-4bb0-bddf-d90d5407d102" />
      <body
        className={`antialiased`}
      >
        {children}
      </body>
    </html>
  );
}
