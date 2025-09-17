'use client'

import { useState, useEffect } from 'react'
import { usePathname, useRouter } from 'next/navigation'
import { Button } from '@/components/ui/button'
import { Sheet, SheetContent, SheetTrigger } from '@/components/ui/sheet'
import LoginPage from '@/components/admin/LoginPage'
import {
  getUserInfo,
  clearAuthInfo,
  isAuthenticated,
  type AuthUser
} from '@/lib/auth'
import Link from 'next/link'
import { Menu } from 'lucide-react'

const navItems = [
  { key: 'endpoints', label: 'API端点', href: '/admin' },
  { key: 'rules', label: 'URL替换规则', href: '/admin/rules' },
  { key: 'home', label: '首页配置', href: '/admin/home' },
  { key: 'stats', label: '域名统计', href: '/admin/stats' },
]

export default function AdminLayout({
  children,
}: {
  children: React.ReactNode
}) {
  const [user, setUser] = useState<AuthUser | null>(null)
  const [loading, setLoading] = useState(true)
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false)
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
      <header className="bg-background shadow-sm border-b sticky top-0 z-50">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center h-16">
            {/* Logo and Desktop Nav */}
            <div className="flex items-center">
              <h1 className="text-lg sm:text-xl font-semibold">
                <Link href="/" className="hover:text-primary transition-colors">
                  随机API管理后台
                </Link>
              </h1>

              {/* Desktop Navigation */}
              <nav className="hidden md:flex md:ml-8 md:space-x-8">
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

            {/* Desktop User Info & Logout */}
            <div className="hidden md:flex md:items-center md:space-x-4">
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

            {/* Mobile Menu Button */}
            <div className="md:hidden">
              <Sheet open={mobileMenuOpen} onOpenChange={setMobileMenuOpen}>
                <SheetTrigger asChild>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="p-2"
                    aria-label="打开菜单"
                  >
                    <Menu className="h-6 w-6" />
                  </Button>
                </SheetTrigger>
                <SheetContent side="right" className="w-[280px] sm:w-[300px]">
                  <div className="flex flex-col h-full">
                    {/* Mobile Header */}
                    <div className="flex items-center justify-between my-6 mx-2">
                      <h2 className="text-lg font-semibold">导航菜单</h2>
                    </div>

                    {/* User Info */}
                    <div className="mb-6 p-4 bg-muted rounded-lg">
                      <p className="text-sm text-muted-foreground mb-1">当前用户</p>
                      <p className="font-medium">{user.name}</p>
                    </div>

                    {/* Mobile Navigation */}
                    <nav className="flex-1">
                      <div className="space-y-2">
                        {navItems.map((item) => (
                          <Link
                            key={item.key}
                            href={item.href}
                            onClick={() => setMobileMenuOpen(false)}
                            className={`flex items-center px-4 py-3 text-sm font-medium rounded-lg transition-colors ${
                              pathname === item.href
                                ? 'bg-primary text-primary-foreground'
                                : 'text-muted-foreground hover:text-foreground hover:bg-muted'
                            }`}
                          >
                            {item.label}
                          </Link>
                        ))}
                      </div>
                    </nav>

                    {/* Mobile Logout */}
                    <div className="pt-4 border-t">
                      <Button
                        onClick={() => {
                          handleLogout()
                          setMobileMenuOpen(false)
                        }}
                        variant="ghost"
                        className="w-full justify-start text-red-600 hover:text-red-700 hover:bg-red-50"
                      >
                        退出登录
                      </Button>
                    </div>
                  </div>
                </SheetContent>
              </Sheet>
            </div>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6 sm:py-8">
        {children}
      </main>
    </div>
  )
} 