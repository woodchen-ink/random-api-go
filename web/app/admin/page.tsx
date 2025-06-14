'use client'

import { useState, useEffect } from 'react'
import EndpointsTab from '@/components/admin/EndpointsTab'
import type { APIEndpoint } from '@/types/admin'
import { authenticatedFetch } from '@/lib/auth'

export default function AdminPage() {
  const [endpoints, setEndpoints] = useState<APIEndpoint[]>([])

  useEffect(() => {
    loadEndpoints()
  }, [])

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

  const createEndpoint = async (endpointData: Partial<APIEndpoint>) => {
    try {
      const response = await authenticatedFetch('/api/admin/endpoints/', {
        method: 'POST',
        body: JSON.stringify(endpointData),
      })

      if (response.ok) {
        loadEndpoints() // 重新加载数据
      } else {
        alert('创建端点失败')
      }
    } catch (error) {
      console.error('Failed to create endpoint:', error)
      alert('创建端点失败')
    }
  }

  return (
    <EndpointsTab 
      endpoints={endpoints} 
      onCreateEndpoint={createEndpoint}
      onUpdateEndpoints={loadEndpoints}
    />
  )
}

 