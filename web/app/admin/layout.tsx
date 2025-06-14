'use client'

import { useState, useEffect } from 'react'
import { usePathname, useRouter } from 'next/navigation'
import { Button } from '@/components/ui/button'
import LoginPage from '@/components/admin/LoginPage'
import { 
  getUserInfo, 
  clearAuthInfo, 
  isAuthenticated,
  type AuthUser
} from '@/lib/auth'
import Link from 'next/link'

const navItems = [
  { key: 'endpoints', label: 'API端点', href: '/admin' },
  { key: 'rules', label: 'URL替换规则', href: '/admin/rules' },
  { key: 'home', label: '首页配置', href: '/admin/home' },
]

export default function AdminLayout({
  children,
}: {
  children: React.ReactNode
}) {
  const [user, setUser] = useState<AuthUser | null>(null)
  const [loading, setLoading] = useState(true)
  const pathname = usePathname()
  const router = useRouter()

  useEffect(() => {
    checkAuth()
  }, [])

  const checkAuth = async () => {
    if (!isAuthenticated()) {
      setLoading(false)
      return
    }

    const savedUser = getUserInfo()
    if (savedUser) {
      setUser(savedUser)
      setLoading(false)
      return
    }

    // 如果没有用户信息，清除认证状态
    clearAuthInfo()
    setLoading(false)
  }

  const handleLoginSuccess = (userInfo: AuthUser) => {
    setUser(userInfo)
    setLoading(false)
  }

  const handleLogout = () => {
    clearAuthInfo()
    setUser(null)
    router.push('/admin')
  }

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="animate-spin rounded-full h-32 w-32 border-b-2 border-primary"></div>
      </div>
    )
  }

  if (!user) {
    return <LoginPage onLoginSuccess={handleLoginSuccess} />
  }

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <header className="bg-background shadow-sm border-b">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center h-16">
            <div className="flex items-center">
              <h1 className="text-xl font-semibold mr-8">
                <Link href="/">
                  随机API管理后台
                </Link>
              </h1>
              
              {/* Navigation */}
              <nav className="flex space-x-8">
                {navItems.map((item) => (
                  <Link
                    key={item.key}
                    href={item.href}
                    className={`px-3 py-2 text-sm font-medium rounded-md transition-colors ${
                      pathname === item.href
                        ? 'bg-primary text-primary-foreground'
                        : 'text-muted-foreground hover:text-foreground hover:bg-muted'
                    }`}
                  >
                    {item.label}
                  </Link>
                ))}
              </nav>
            </div>
            
            <div className="flex items-center space-x-4">
              <span className="text-sm text-muted-foreground">
                欢迎, {user.name}
              </span>
              <Button
                onClick={handleLogout}
                variant="ghost"
                size="sm"
                className="text-red-600 hover:text-red-700"
              >
                退出登录
              </Button>
            </div>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {children}
      </main>
    </div>
  )
} 