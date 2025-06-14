'use client'

import { useState, useEffect } from 'react'
import HomeConfigTab from '@/components/admin/HomeConfigTab'
import { authenticatedFetch } from '@/lib/auth'

export default function HomePage() {
  const [homeConfig, setHomeConfig] = useState('')

  useEffect(() => {
    loadHomeConfig()
  }, [])

  const loadHomeConfig = async () => {
    try {
      const response = await authenticatedFetch('/api/admin/home-config')
      if (response.ok) {
        const data = await response.json()
        setHomeConfig(data.data?.content || '')
      }
    } catch (error) {
      console.error('Failed to load home config:', error)
    }
  }

  const updateHomeConfig = async (content: string) => {
    try {
      const response = await authenticatedFetch('/api/admin/home-config', {
        method: 'POST',
        body: JSON.stringify({ content }),
      })

      if (response.ok) {
        alert('首页配置更新成功')
        setHomeConfig(content) // 更新本地状态
      } else {
        alert('首页配置更新失败')
      }
    } catch (error) {
      console.error('Failed to update home config:', error)
      alert('首页配置更新失败')
    }
  }

  return (
    <HomeConfigTab config={homeConfig} onUpdate={updateHomeConfig} />
  )
} 