import Cookies from 'js-cookie'

const TOKEN_COOKIE_NAME = 'admin_token'
const USER_INFO_COOKIE_NAME = 'admin_user'

// Cookie配置
const COOKIE_OPTIONS = {
  expires: 7, // 7天过期
  secure: process.env.NODE_ENV === 'production', // 生产环境使用HTTPS
  sameSite: 'strict' as const,
  path: '/'
}

export interface AuthUser {
  id: string
  name: string
  email: string
}

// 保存认证信息
export function saveAuthInfo(token: string, user: AuthUser) {
  Cookies.set(TOKEN_COOKIE_NAME, token, COOKIE_OPTIONS)
  Cookies.set(USER_INFO_COOKIE_NAME, JSON.stringify(user), COOKIE_OPTIONS)
}

// 获取访问令牌
export function getAccessToken(): string | null {
  return Cookies.get(TOKEN_COOKIE_NAME) || null
}

// 获取用户信息
export function getUserInfo(): AuthUser | null {
  const userStr = Cookies.get(USER_INFO_COOKIE_NAME)
  if (!userStr) return null
  
  try {
    return JSON.parse(userStr)
  } catch {
    return null
  }
}

// 清除认证信息
export function clearAuthInfo() {
  Cookies.remove(TOKEN_COOKIE_NAME, { path: '/' })
  Cookies.remove(USER_INFO_COOKIE_NAME, { path: '/' })
}

// 检查是否已登录
export function isAuthenticated(): boolean {
  return !!getAccessToken()
}

// 创建带认证的fetch请求
export async function authenticatedFetch(url: string, options: RequestInit = {}): Promise<Response> {
  const token = getAccessToken()
  
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(options.headers as Record<string, string> || {}),
  }
  
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }
  
  const response = await fetch(url, {
    ...options,
    headers,
  })
  
  // 如果token过期或无效，清除认证信息并重定向到登录
  if (response.status === 401) {
    clearAuthInfo()
    // 可以选择重定向到登录页面或显示登录提示
    if (typeof window !== 'undefined') {
      window.location.href = '/admin'
    }
  }
  
  return response
}

// OAuth状态管理
export function saveOAuthState(state: string) {
  sessionStorage.setItem('oauth_state', state)
}

export function getOAuthState(): string | null {
  return sessionStorage.getItem('oauth_state')
}

export function clearOAuthState() {
  sessionStorage.removeItem('oauth_state')
} 