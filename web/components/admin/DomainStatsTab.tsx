'use client'

import { useState, useEffect, useRef, useCallback } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Switch } from '@/components/ui/switch'
import { Button } from '@/components/ui/button'
import { authenticatedFetch } from '@/lib/auth'
import type { DomainStatsResult } from '@/types/admin'

// 表格组件，用于显示域名统计数据
const DomainStatsTable = ({ 
  title, 
  data, 
  loading 
}: { 
  title: string; 
  data: DomainStatsResult[] | null; 
  loading: boolean 
}) => {
  const formatNumber = (num: number) => {
    return num.toLocaleString()
  }

  if (loading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>{title}</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex justify-center items-center py-8">
            <div className="text-gray-500">加载中...</div>
          </div>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>{title}</CardTitle>
      </CardHeader>
      <CardContent>
        {!data || data.length === 0 ? (
          <p className="text-gray-500 text-center py-4">暂无数据</p>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-12">排名</TableHead>
                <TableHead>域名</TableHead>
                <TableHead className="text-right">访问次数</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {data.map((item, index) => (
                <TableRow key={`${item.domain}-${item.count}`}>
                  <TableCell className="font-medium">{index + 1}</TableCell>
                  <TableCell>
                    <span className="font-mono">
                      {item.domain === 'direct' ? '直接访问' : 
                       item.domain === 'unknown' ? '未知来源' : 
                       item.domain}
                    </span>
                  </TableCell>
                  <TableCell className="text-right">
                    {formatNumber(item.count)}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </CardContent>
    </Card>
  )
}

export default function DomainStatsTab() {
  // 状态管理
  const [loading, setLoading] = useState(false)
  const [refreshing, setRefreshing] = useState(false)
  const [stats24h, setStats24h] = useState<DomainStatsResult[] | null>(null)
  const [statsTotal, setStatsTotal] = useState<DomainStatsResult[] | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [autoRefresh, setAutoRefresh] = useState(true)
  const [lastUpdateTime, setLastUpdateTime] = useState<Date | null>(null)
  const intervalRef = useRef<NodeJS.Timeout | null>(null)

  // 加载域名统计数据
  const loadDomainStats = useCallback(async (isInitialLoad = false) => {
    try {
      if (isInitialLoad) {
        setLoading(true)
      } else {
        setRefreshing(true)
      }
      setError(null)
      
      const response = await authenticatedFetch('/api/admin/domain-stats')
      if (response.ok) {
        const data = await response.json()
        if (data.data) {
          setStats24h(data.data.top_24_hours || [])
          setStatsTotal(data.data.top_total || [])
          setLastUpdateTime(new Date())
        }
      } else {
        throw new Error('获取域名统计失败')
      }
    } catch (error) {
      console.error('Failed to load domain stats:', error)
      setError('获取域名统计失败')
    } finally {
      setLoading(false)
      setRefreshing(false)
    }
  }, [])

  // 初始加载
  useEffect(() => {
    loadDomainStats(true)
  }, [loadDomainStats])

  // 自动刷新设置
  useEffect(() => {
    if (autoRefresh) {
      // 设置自动刷新定时器
      intervalRef.current = setInterval(() => {
        loadDomainStats(false)
      }, 5000) // 每5秒刷新一次
    } else {
      // 清除定时器
      if (intervalRef.current) {
        clearInterval(intervalRef.current)
        intervalRef.current = null
      }
    }

    // 清理函数
    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current)
      }
    }
  }, [autoRefresh, loadDomainStats])

  // 格式化更新时间
  const formatUpdateTime = (time: Date | null) => {
    if (!time) return ''
    return time.toLocaleTimeString()
  }

  // 显示错误状态
  if (error && !stats24h && !statsTotal) {
    return (
      <div className="flex flex-col items-center justify-center py-8 space-y-4">
        <div className="text-red-500">{error}</div>
        <Button
          onClick={() => loadDomainStats(true)}
          variant="default"
        >
          重试
        </Button>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h2 className="text-2xl font-bold">域名访问统计</h2>
          {lastUpdateTime && (
            <p className="text-sm text-gray-500 mt-1">
              最后更新: {formatUpdateTime(lastUpdateTime)}
              {refreshing && <span className="ml-2 animate-pulse">刷新中...</span>}
            </p>
          )}
        </div>
        <div className="flex items-center space-x-4">
          <div className="flex items-center space-x-2">
            <Switch
              checked={autoRefresh}
              onCheckedChange={setAutoRefresh}
              id="auto-refresh"
            />
            <label htmlFor="auto-refresh" className="text-sm">
              自动刷新 (5秒)
            </label>
          </div>
          <Button
            onClick={() => loadDomainStats(false)}
            disabled={refreshing}
            variant="default"
          >
            {refreshing ? '刷新中...' : '手动刷新'}
          </Button>
        </div>
      </div>

      <div className="grid gap-6 md:grid-cols-2">
        <DomainStatsTable 
          title="24小时内访问最多的域名 (前30)" 
          data={stats24h} 
          loading={loading && !stats24h} 
        />
        
        <DomainStatsTable 
          title="总访问最多的域名 (前30)" 
          data={statsTotal} 
          loading={loading && !statsTotal} 
        />
      </div>
    </div>
  )
} 