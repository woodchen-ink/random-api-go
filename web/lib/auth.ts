import Cookies from 'js-cookie'

const TOKEN_COOKIE_NAME = 'admin_token'
const REFRESH_TOKEN_COOKIE_NAME = 'admin_refresh_token'
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
export function saveAuthInfo(token: string, user: AuthUser, refreshToken?: string) {
  Cookies.set(TOKEN_COOKIE_NAME, token, COOKIE_OPTIONS)
  Cookies.set(USER_INFO_COOKIE_NAME, JSON.stringify(user), COOKIE_OPTIONS)
  
  if (refreshToken) {
    Cookies.set(REFRESH_TOKEN_COOKIE_NAME, refreshToken, COOKIE_OPTIONS)
  }
}

// 获取访问令牌
export function getAccessToken(): string | null {
  return Cookies.get(TOKEN_COOKIE_NAME) || null
}

// 获取刷新令牌
export function getRefreshToken(): string | null {
  return Cookies.get(REFRESH_TOKEN_COOKIE_NAME) || null
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
  Cookies.remove(REFRESH_TOKEN_COOKIE_NAME, { path: '/' })
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
    credentials: 'include', // 包含cookie
  })
  
  // 如果token过期，尝试刷新
  if (response.status === 401) {
    const refreshed = await refreshAccessToken()
    if (refreshed) {
      // 重新发送请求
      const newToken = getAccessToken()
      if (newToken) {
        headers['Authorization'] = `Bearer ${newToken}`
        return fetch(url, {
          ...options,
          headers,
          credentials: 'include',
        })
      }
    }
    // 刷新失败，清除认证信息
    clearAuthInfo()
  }
  
  return response
}

// 刷新访问令牌
async function refreshAccessToken(): Promise<boolean> {
  const refreshToken = getRefreshToken()
  if (!refreshToken) return false
  
  try {
    const response = await fetch('/api/admin/auth/refresh', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ refresh_token: refreshToken }),
      credentials: 'include',
    })
    
    if (response.ok) {
      const data = await response.json()
      if (data.success && data.data.access_token) {
        const user = getUserInfo()
        if (user) {
          saveAuthInfo(data.data.access_token, user, data.data.refresh_token)
          return true
        }
      }
    }
  } catch (error) {
    console.error('Failed to refresh token:', error)
  }
  
  return false
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