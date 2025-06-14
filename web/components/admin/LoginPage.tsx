'use client'

import { useState, useEffect } from 'react'
import { Button } from '@/components/ui/button'
import { 
  saveAuthInfo, 
  saveOAuthState,
  getOAuthState,
  clearOAuthState,
  type AuthUser
} from '@/lib/auth'
import type { OAuthConfig } from '@/types/admin'

// OAuth2.0 端点配置
const OAUTH_ENDPOINTS = {
  authorizeUrl: 'https://connect.czl.net/oauth2/authorize',
  tokenUrl: 'https://connect.czl.net/api/oauth2/token',
  userInfoUrl: 'https://connect.czl.net/api/oauth2/userinfo',
  // 使用配置的BASE_URL构建回调地址
  getRedirectUri: (baseUrl: string) => {
    return `${baseUrl}/api/admin/oauth/callback`
  }
}

interface LoginPageProps {
  onLoginSuccess: (user: AuthUser) => void
}

export default function LoginPage({ onLoginSuccess }: LoginPageProps) {
  const [loading, setLoading] = useState(true)
  const [oauthConfig, setOauthConfig] = useState<OAuthConfig | null>(null)

  useEffect(() => {
    // 首先检查URL参数中是否有token
    checkURLParams()
    loadOAuthConfig()
  }, [])

  const checkURLParams = () => {
    const urlParams = new URLSearchParams(window.location.search)
    const token = urlParams.get('token')
    const error = urlParams.get('error')
    const userName = urlParams.get('user')
    const state = urlParams.get('state')

    if (error) {
      alert(`登录失败: ${error}`)
      // 清理URL参数
      window.history.replaceState({}, document.title, window.location.pathname)
      return
    }

    if (token) {
      // 验证state参数防止CSRF攻击（如果存在的话）
      if (state) {
        const savedState = getOAuthState()
        if (savedState !== state) {
          alert('登录状态验证失败，请重新登录')
          clearOAuthState()
          window.history.replaceState({}, document.title, window.location.pathname)
          return
        }
      } else {
        console.warn('OAuth回调缺少state参数，可能存在安全风险')
      }

      // 保存认证信息
      if (userName) {
        const userInfo: AuthUser = { id: '', name: userName, email: '' }
        saveAuthInfo(token, userInfo)
        onLoginSuccess(userInfo)
      }
      
      // 清理URL参数和OAuth状态
      clearOAuthState()
      window.history.replaceState({}, document.title, window.location.pathname)
      return
    }

    setLoading(false)
  }

  const loadOAuthConfig = async () => {
    try {
      console.log('Loading OAuth config...')
      const response = await fetch('/api/oauth-config')
      console.log('OAuth config response status:', response.status)
      
      if (response.ok) {
        const data = await response.json()
        console.log('OAuth config data:', data)
        if (data.success) {
          setOauthConfig(data.data)
        } else {
          // OAuth配置错误
          console.error('OAuth配置错误:', data.error)
          alert(`OAuth配置错误: ${data.error}`)
        }
      } else {
        const errorText = await response.text()
        console.error('Failed to load OAuth config: HTTP', response.status, errorText)
        alert(`无法加载OAuth配置: HTTP ${response.status}`)
      }
    } catch (error) {
      console.error('Failed to load OAuth config:', error)
      alert(`网络错误: ${error instanceof Error ? error.message : '未知错误'}`)
    } finally {
      setLoading(false)
    }
  }

  const handleLogin = () => {
    if (!oauthConfig) {
      alert('OAuth配置未加载')
      return
    }

    // 生成随机state值防止CSRF攻击
    const state = Math.random().toString(36).substring(2, 15) + Math.random().toString(36).substring(2, 15)
    saveOAuthState(state)

    const params = new URLSearchParams({
      client_id: oauthConfig.client_id,
      redirect_uri: OAUTH_ENDPOINTS.getRedirectUri(oauthConfig.base_url), // 使用配置的BASE_URL
      response_type: 'code',
      scope: 'read write', // 根据CZL Connect文档使用正确的scope
      state: state,
    })
    
    window.location.href = `${OAUTH_ENDPOINTS.authorizeUrl}?${params.toString()}`
  }

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="animate-spin rounded-full h-32 w-32 border-b-2 border-primary"></div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-background flex items-center justify-center">
      <div className="bg-card rounded-lg border shadow-lg p-8 max-w-md w-full mx-4">
        <div className="text-center mb-8">
          <div className="inline-flex items-center justify-center w-16 h-16 bg-primary rounded-full mb-4">
            <svg className="w-8 h-8 text-primary-foreground" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
            </svg>
          </div>
          <h1 className="text-2xl font-bold mb-2">
            管理后台登录
          </h1>
          <p className="text-muted-foreground">
            请使用 CZL Connect 账号登录
          </p>
        </div>
        
        <Button
          onClick={handleLogin}
          className="w-full"
          size="lg"
          disabled={!oauthConfig}
        >
          {oauthConfig ? '使用 CZL Connect 登录' : '加载中...'}
        </Button>
      </div>
    </div>
  )
} 