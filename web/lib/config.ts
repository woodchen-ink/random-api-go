// 应用配置管理

export interface AppConfig {
  apiBaseUrl: string
  isProduction: boolean
  isDevelopment: boolean
}

// 获取API基础URL
export function getApiBaseUrl(): string {
  // 在服务端渲染时
  if (typeof window === 'undefined') {
    // 1. 优先使用环境变量
    if (process.env.BASE_URL) {
      return process.env.BASE_URL
    }
    
    // 2. 生产环境使用相对路径（假设前后端部署在同一域名）
    if (process.env.NODE_ENV === 'production') {
      return ''
    }
    
    // 3. 开发环境默认值
    return 'http://localhost:5003'
  }
  
  // 在客户端，使用相对路径（自动使用当前域名和端口）
  return ''
}

// 获取完整的API URL
export function getApiUrl(path: string): string {
  const baseUrl = getApiBaseUrl()
  const cleanPath = path.startsWith('/') ? path : `/${path}`
  return `${baseUrl}${cleanPath}`
}

// 获取应用配置
export function getAppConfig(): AppConfig {
  return {
    apiBaseUrl: getApiBaseUrl(),
    isProduction: process.env.NODE_ENV === 'production',
    isDevelopment: process.env.NODE_ENV === 'development',
  }
}

// 创建带有默认配置的fetch函数
export async function apiFetch(path: string, options: RequestInit = {}): Promise<Response> {
  const url = getApiUrl(path)
  
  const defaultOptions: RequestInit = {
    headers: {
      'Content-Type': 'application/json',
      ...options.headers,
    },
    cache: 'no-store', // 默认不缓存
    ...options,
  }
  
  return fetch(url, defaultOptions)
} 