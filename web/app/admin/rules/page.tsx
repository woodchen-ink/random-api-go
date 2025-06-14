'use client'

import { useState, useEffect } from 'react'
import URLRulesTab from '@/components/admin/URLRulesTab'
import type { URLReplaceRule, APIEndpoint } from '@/types/admin'
import { authenticatedFetch } from '@/lib/auth'

export default function RulesPage() {
  const [urlRules, setUrlRules] = useState<URLReplaceRule[]>([])
  const [endpoints, setEndpoints] = useState<APIEndpoint[]>([])

  useEffect(() => {
    loadURLRules()
    loadEndpoints()
  }, [])

  const loadURLRules = async () => {
    try {
      const response = await authenticatedFetch('/api/admin/url-replace-rules')
      if (response.ok) {
        const data = await response.json()
        setUrlRules(data.data || [])
      }
    } catch (error) {
      console.error('Failed to load URL rules:', error)
    }
  }

  const loadEndpoints = async () => {
    try {
      const response = await authenticatedFetch('/api/admin/endpoints')
      if (response.ok) {
        const data = await response.json()
        setEndpoints(data.data || [])
      }
    } catch (error) {
      console.error('Failed to load endpoints:', error)
    }
  }

  const createURLRule = async (ruleData: Partial<URLReplaceRule>) => {
    try {
      const response = await authenticatedFetch('/api/admin/url-replace-rules', {
        method: 'POST',
        body: JSON.stringify(ruleData),
      })

      if (response.ok) {
        loadURLRules() // 重新加载数据
        alert('URL替换规则创建成功')
      } else {
        alert('创建URL替换规则失败')
      }
    } catch (error) {
      console.error('Failed to create URL rule:', error)
      alert('创建URL替换规则失败')
    }
  }

  const updateURLRule = async (id: number, ruleData: Partial<URLReplaceRule>) => {
    try {
      const response = await authenticatedFetch(`/api/admin/url-replace-rules/${id}`, {
        method: 'PUT',
        body: JSON.stringify(ruleData),
      })

      if (response.ok) {
        loadURLRules() // 重新加载数据
        alert('URL替换规则更新成功')
      } else {
        alert('更新URL替换规则失败')
      }
    } catch (error) {
      console.error('Failed to update URL rule:', error)
      alert('更新URL替换规则失败')
    }
  }

  const deleteURLRule = async (id: number) => {
    try {
      const response = await authenticatedFetch(`/api/admin/url-replace-rules/${id}`, {
        method: 'DELETE',
      })

      if (response.ok) {
        loadURLRules() // 重新加载数据
        alert('URL替换规则删除成功')
      } else {
        alert('删除URL替换规则失败')
      }
    } catch (error) {
      console.error('Failed to delete URL rule:', error)
      alert('删除URL替换规则失败')
    }
  }

  return (
    <URLRulesTab 
      rules={urlRules} 
      endpoints={endpoints}
      onCreateRule={createURLRule}
      onUpdateRule={updateURLRule}
      onDeleteRule={deleteURLRule}
    />
  )
} 