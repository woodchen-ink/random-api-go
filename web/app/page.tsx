'use client'

import { useState, useEffect } from 'react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'

import { Button } from '@/components/ui/button'
import Link from 'next/link'
import { apiFetch } from '@/lib/config'

interface Endpoint {
  id: number;
  name: string;
  url: string;
  description?: string;
  is_active: boolean;
  show_on_homepage: boolean;
  sort_order: number;
}

interface SystemMetrics {
  uptime: number; // 纳秒
  start_time: string;
  num_cpu: number;
  num_goroutine: number;
  average_latency: number;
  memory_stats: {
    heap_alloc: number;
    heap_sys: number;
  };
}


async function getHomePageConfig() {
  try {
    const res = await apiFetch('/api/admin/home-config')
    if (!res.ok) {
      throw new Error('Failed to fetch home page config')
    }
    const data = await res.json()
    return data.data?.content || '# 欢迎使用随机API服务\n\n服务正在启动中...'
  } catch (error) {
    console.error('Error fetching home page config:', error)
    return '# 欢迎使用随机API服务\n\n这是一个可配置的随机API服务。'
  }
}

async function getStats() {
  try {
    const res = await apiFetch('/api/stats')
    if (!res.ok) {
      throw new Error('Failed to fetch stats')
    }
    return await res.json()
  } catch (error) {
    console.error('Error fetching stats:', error)
    return {}
  }
}

async function getURLStats() {
  try {
    const res = await apiFetch('/api/urlstats')
    if (!res.ok) {
      throw new Error('Failed to fetch URL stats')
    }
    return await res.json()
  } catch (error) {
    console.error('Error fetching URL stats:', error)
    return {}
  }
}

async function getSystemMetrics(): Promise<SystemMetrics | null> {
  try {
    const res = await apiFetch('/api/metrics')
    if (!res.ok) {
      throw new Error('Failed to fetch system metrics')
    }
    return await res.json()
  } catch (error) {
    console.error('Error fetching system metrics:', error)
    return null
  }
}

async function getEndpoints() {
  try {
    const res = await apiFetch('/api/admin/endpoints')
    if (!res.ok) {
      throw new Error('Failed to fetch endpoints')
    }
    const data = await res.json()
    return data.data || []
  } catch (error) {
    console.error('Error fetching endpoints:', error)
    return []
  }
}



function formatUptime(uptimeNs: number): string {
  const uptimeMs = uptimeNs / 1000000; // 纳秒转毫秒
  const days = Math.floor(uptimeMs / (1000 * 60 * 60 * 24));
  const hours = Math.floor((uptimeMs % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60));
  const minutes = Math.floor((uptimeMs % (1000 * 60 * 60)) / (1000 * 60));
  
  if (days > 0) {
    return `${days}天 ${hours}小时 ${minutes}分钟`;
  } else if (hours > 0) {
    return `${hours}小时 ${minutes}分钟`;
  } else {
    return `${minutes}分钟`;
  }
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

function formatStartTime(startTime: string): string {
  const date = new Date(startTime);
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  });
}

// 复制到剪贴板的函数
function copyToClipboard(text: string) {
  if (navigator.clipboard && window.isSecureContext) {
    return navigator.clipboard.writeText(text);
  } else {
    // 降级方案 - 抑制弃用警告
    const textArea = document.createElement('textarea');
    textArea.value = text;
    textArea.style.position = 'fixed';
    textArea.style.left = '-999999px';
    textArea.style.top = '-999999px';
    document.body.appendChild(textArea);
    textArea.focus();
    textArea.select();
    return new Promise<void>((resolve, reject) => {
              try {
          const success = document.execCommand('copy');
        if (success) {
          resolve();
        } else {
          reject(new Error('Copy command failed'));
        }
      } catch (err) {
        reject(err);
      } finally {
        textArea.remove();
      }
    });
  }
}

export default function Home() {
  const [content, setContent] = useState('')
  const [stats, setStats] = useState<{Stats?: Record<string, {TotalCalls: number, TodayCalls: number}>}>({})
  const [urlStats, setUrlStats] = useState<Record<string, {total_urls: number}>>({})
  const [systemMetrics, setSystemMetrics] = useState<SystemMetrics | null>(null)
  const [endpoints, setEndpoints] = useState<Endpoint[]>([])
  const [copiedUrl, setCopiedUrl] = useState<string | null>(null)

  useEffect(() => {
    const loadData = async () => {
      const [contentData, statsData, urlStatsData, systemMetricsData, endpointsData] = await Promise.all([
        getHomePageConfig(),
        getStats(),
        getURLStats(),
        getSystemMetrics(),
        getEndpoints()
      ])
      
      setContent(contentData)
      setStats(statsData)
      setUrlStats(urlStatsData)
      setSystemMetrics(systemMetricsData)
      setEndpoints(endpointsData)
    }

    loadData()
  }, [])
  


  // 过滤出首页可见的端点
  const visibleEndpoints = endpoints.filter((endpoint: Endpoint) => 
    endpoint.is_active && endpoint.show_on_homepage
  )

  const handleCopyUrl = async (endpoint: Endpoint) => {
    const fullUrl = `${window.location.origin}/${endpoint.url}`
    try {
      await copyToClipboard(fullUrl)
      setCopiedUrl(endpoint.url)
      setTimeout(() => setCopiedUrl(null), 2000) // 2秒后清除复制状态
    } catch (err) {
      console.error('复制失败:', err)
    }
  }

  return (
    <div 
      className="min-h-screen bg-gray-100 dark:bg-gray-900 relative"
      style={{
        backgroundImage: 'url(http://localhost:5003/pic/all)',
        backgroundSize: 'cover',
        backgroundPosition: 'center',
        backgroundAttachment: 'fixed'
      }}
    >
      {/* 背景遮罩 */}
      <div className="absolute inset-0 bg-white/70 dark:bg-black/70 backdrop-blur-sm"></div>
      
      <div className="container mx-auto px-4 py-6 relative z-10">
        <div className="max-w-7xl mx-auto">
          {/* Header - 更简洁 */}
          <div className="text-center mb-6">
            <div className="inline-flex items-center justify-center w-12 h-12 bg-gray-800 dark:bg-gray-200 rounded-full mb-3">
              <svg className="w-6 h-6 text-white dark:text-gray-800" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
              </svg>
            </div>
          </div>

          {/* System Status Section - 性冷淡风格 */}
          {systemMetrics && (
            <div className="mb-6">
              <h2 className="text-xl font-bold mb-4 text-gray-800 dark:text-gray-200 text-center">
                系统状态
              </h2>
              <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-5 gap-3">
                {/* 运行时间 */}
                <div className="bg-white/80 dark:bg-gray-800/80 backdrop-blur-sm rounded-lg shadow-sm border border-gray-200/50 dark:border-gray-700/50 p-4">
                  <div className="flex items-center justify-between mb-2">
                    <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300">运行时间</h3>
                    <div className="w-2 h-2 bg-gray-400 dark:bg-gray-500 rounded-full"></div>
                  </div>
                  <p className="text-lg font-bold text-gray-900 dark:text-gray-100 leading-tight">
                    {formatUptime(systemMetrics.uptime)}
                  </p>
                  <p className="text-xs text-gray-500 dark:text-gray-400 mt-1 truncate">
                    {formatStartTime(systemMetrics.start_time)}
                  </p>
                </div>

                {/* CPU核心数 */}
                <div className="bg-white/80 dark:bg-gray-800/80 backdrop-blur-sm rounded-lg shadow-sm border border-gray-200/50 dark:border-gray-700/50 p-4">
                  <div className="flex items-center justify-between mb-2">
                    <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300">CPU核心</h3>
                    <svg className="w-4 h-4 text-gray-500 dark:text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z" />
                    </svg>
                  </div>
                  <p className="text-xl font-bold text-gray-900 dark:text-gray-100">
                    {systemMetrics.num_cpu} 核
                  </p>
                </div>

                {/* Goroutine数量 */}
                <div className="bg-white/80 dark:bg-gray-800/80 backdrop-blur-sm rounded-lg shadow-sm border border-gray-200/50 dark:border-gray-700/50 p-4">
                  <div className="flex items-center justify-between mb-2">
                    <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300">协程数</h3>
                    <svg className="w-4 h-4 text-gray-500 dark:text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
                    </svg>
                  </div>
                  <p className="text-xl font-bold text-gray-900 dark:text-gray-100">
                    {systemMetrics.num_goroutine}
                  </p>
                </div>

                {/* 平均延迟 */}
                <div className="bg-white/80 dark:bg-gray-800/80 backdrop-blur-sm rounded-lg shadow-sm border border-gray-200/50 dark:border-gray-700/50 p-4">
                  <div className="flex items-center justify-between mb-2">
                    <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300">平均延迟</h3>
                    <svg className="w-4 h-4 text-gray-500 dark:text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                    </svg>
                  </div>
                  <p className="text-xl font-bold text-gray-900 dark:text-gray-100">
                    {systemMetrics.average_latency.toFixed(2)} ms
                  </p>
                </div>

                {/* 堆内存分配 */}
                <div className="bg-white/80 dark:bg-gray-800/80 backdrop-blur-sm rounded-lg shadow-sm border border-gray-200/50 dark:border-gray-700/50 p-4">
                  <div className="flex items-center justify-between mb-2">
                    <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300">堆内存</h3>
                    <svg className="w-4 h-4 text-gray-500 dark:text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v4a2 2 0 01-2 2H9a2 2 0 01-2-2z" />
                    </svg>
                  </div>
                  <p className="text-lg font-bold text-gray-900 dark:text-gray-100">
                    {formatBytes(systemMetrics.memory_stats.heap_alloc)}
                  </p>
                  <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                    系统: {formatBytes(systemMetrics.memory_stats.heap_sys)}
                  </p>
                </div>

                {/* 当前时间
                <div className="bg-white/80 dark:bg-gray-800/80 backdrop-blur-sm rounded-lg shadow-sm border border-gray-200/50 dark:border-gray-700/50 p-4">
                  <div className="flex items-center justify-between mb-2">
                    <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300">当前时间</h3>
                    <svg className="w-4 h-4 text-gray-500 dark:text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
                    </svg>
                  </div>
                  <p className="text-sm font-bold text-gray-900 dark:text-gray-100 leading-tight">
                    {new Date().toLocaleString('zh-CN', {
                      month: '2-digit',
                      day: '2-digit',
                      hour: '2-digit',
                      minute: '2-digit'
                    })}
                  </p>
                </div> */}
              </div>
            </div>
          )}

          

          {/* API端点统计 - 全宽布局 */}
          {visibleEndpoints.length > 0 && (
            <div className="mb-6">
              <h2 className="text-xl font-bold mb-4 text-gray-800 dark:text-gray-200">
                API 端点统计
              </h2>
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
                {visibleEndpoints.map((endpoint: Endpoint) => {
                  const endpointStats = stats.Stats?.[endpoint.url] || { TotalCalls: 0, TodayCalls: 0 }
                  const urlCount = urlStats[endpoint.url]?.total_urls || 0
                  
                  return (
                    <div key={endpoint.id} className="bg-white/80 dark:bg-gray-800/80 backdrop-blur-sm rounded-lg shadow-sm border border-gray-200/50 dark:border-gray-700/50 p-4">
                      <div className="flex items-start justify-between mb-3">
                        <div className="flex-1 min-w-0">
                          <h3 className="text-base font-semibold text-gray-900 dark:text-gray-100 truncate">
                            {endpoint.name}
                          </h3>
                          <p className="text-xs text-gray-500 dark:text-gray-400 mt-1 font-mono truncate">
                            /{endpoint.url}
                          </p>
                        </div>
                        <div className="flex items-center space-x-2 ml-2">
                          <Button 
                            size="sm" 
                            variant="outline" 
                            className="text-xs px-2 py-1 border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700"
                            onClick={() => handleCopyUrl(endpoint)}
                          >
                            {copiedUrl === endpoint.url ? (
                              <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                              </svg>
                            ) : (
                              <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                              </svg>
                            )}
                          </Button>
                          <Button size="sm" variant="outline" className="text-xs px-2 py-1 border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700">
                            <Link href={`/${endpoint.url}`} target="_blank">
                              访问
                            </Link>
                          </Button>
                        </div>
                      </div>
                      
                      <div className="grid grid-cols-3 gap-3 text-center mb-3">
                        <div className="bg-gray-50/50 dark:bg-gray-700/50 rounded-md p-2">
                          <p className="text-xs text-gray-600 dark:text-gray-400">今日</p>
                          <p className="text-lg font-bold text-gray-900 dark:text-gray-100">
                            {endpointStats.TodayCalls}
                          </p>
                        </div>
                        <div className="bg-gray-50/50 dark:bg-gray-700/50 rounded-md p-2">
                          <p className="text-xs text-gray-600 dark:text-gray-400">总计</p>
                          <p className="text-lg font-bold text-gray-900 dark:text-gray-100">
                            {endpointStats.TotalCalls}
                          </p>
                        </div>
                        <div className="bg-gray-50/50 dark:bg-gray-700/50 rounded-md p-2">
                          <p className="text-xs text-gray-600 dark:text-gray-400">URL</p>
                          <p className="text-lg font-bold text-gray-900 dark:text-gray-100">
                            {urlCount}
                          </p>
                        </div>
                      </div>
                      
                      {endpoint.description && (
                        <p className="text-xs text-gray-500 dark:text-gray-400 line-clamp-2">
                          {endpoint.description}
                        </p>
                      )}
                    </div>
                  )
                })}
              </div>
            </div>
          )}

          {/* Main Content - 半透明 */}
          <div className="bg-white/80 dark:bg-gray-800/80 backdrop-blur-sm rounded-xl shadow-sm border border-gray-200/50 dark:border-gray-700/50 p-6 mb-6">
            <div className="prose prose-sm max-w-none dark:prose-invert">
              <ReactMarkdown 
                remarkPlugins={[remarkGfm]}
                components={{
                  h1: ({children}) => <h1 className="text-3xl font-bold mb-4 text-gray-900 dark:text-white">{children}</h1>,
                  h2: ({children}) => <h2 className="text-xl font-semibold mb-3 text-gray-800 dark:text-gray-200">{children}</h2>,
                  h3: ({children}) => <h3 className="text-lg font-medium mb-2 text-gray-700 dark:text-gray-300">{children}</h3>,
                  p: ({children}) => <p className="mb-3 text-gray-600 dark:text-gray-400">{children}</p>,
                  ul: ({children}) => <ul className="list-disc list-inside mb-3 space-y-1">{children}</ul>,
                  ol: ({children}) => <ol className="list-decimal list-inside mb-3 space-y-1">{children}</ol>,
                  li: ({children}) => <li className="mb-1 text-gray-600 dark:text-gray-400">{children}</li>,
                  strong: ({children}) => <strong className="font-semibold text-gray-900 dark:text-white">{children}</strong>,
                  em: ({children}) => <em className="italic">{children}</em>,
                  code: ({children}) => <code className="bg-gray-100 dark:bg-gray-800 px-2 py-1 rounded text-sm font-mono">{children}</code>,
                  pre: ({children}) => <pre className="bg-gray-100 dark:bg-gray-800 p-4 rounded-lg overflow-x-auto mb-4">{children}</pre>,
                  blockquote: ({children}) => <blockquote className="border-l-4 border-gray-300 dark:border-gray-600 pl-4 italic text-gray-700 dark:text-gray-300 mb-4">{children}</blockquote>,
                  a: ({href, children}) => <a href={href} className="text-blue-600 dark:text-blue-400 hover:underline" target="_blank" rel="noopener noreferrer">{children}</a>,
                }}
              >
                {content}
              </ReactMarkdown>
            </div>
          </div>

          {/* Footer - 包含管理后台链接 */}
          <div className="text-center mt-8 text-sm text-gray-600 dark:text-gray-400">
            <p>随机API服务 - 基于 Next.js 和 Go 构建</p>
            <p className="mt-2">
              <Link href="/admin" className="text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300 underline">
                管理后台
              </Link>
            </p>
          </div>
        </div>
      </div>
    </div>
  )
}
