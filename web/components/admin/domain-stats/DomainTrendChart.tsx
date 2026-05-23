'use client'

import { useEffect, useMemo, useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { authenticatedFetch } from '@/lib/auth'
import { Area, AreaChart, CartesianGrid, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts'
import type { DomainTrendData } from '@/types/admin'

type DaysOption = '7' | '14' | '30' | '60' | '90'

// 趋势卡片: 折线/面积图展示最近 N 天每日总访问量, 不在 sub-tab 切换时重新获取
export default function DomainTrendChart() {
  const [days, setDays] = useState<DaysOption>('30')
  const [loading, setLoading] = useState(false)
  const [data, setData] = useState<DomainTrendData | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    const fetchTrend = async () => {
      try {
        setLoading(true)
        setError(null)
        const resp = await authenticatedFetch(`/api/admin/domain-stats/trend?days=${days}`)
        if (!resp.ok) throw new Error('请求失败')
        const json = await resp.json()
        if (!cancelled) setData(json.data as DomainTrendData)
      } catch (e) {
        if (!cancelled) setError((e as Error).message)
      } finally {
        if (!cancelled) setLoading(false)
      }
    }
    fetchTrend()
    return () => {
      cancelled = true
    }
  }, [days])

  const series = useMemo(() => (data?.series ?? []).map(p => ({
    date: p.date.slice(5),
    count: p.count,
  })), [data])

  const total = useMemo(() => series.reduce((s, p) => s + p.count, 0), [series])
  const peak = useMemo(() => series.reduce((m, p) => (p.count > m ? p.count : m), 0), [series])

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <div>
          <CardTitle className="text-base">每日访问趋势</CardTitle>
          <p className="text-sm text-muted-foreground mt-1">
            合计 {total.toLocaleString()} 次, 峰值 {peak.toLocaleString()} 次/天
          </p>
        </div>
        <Select value={days} onValueChange={(v) => setDays(v as DaysOption)}>
          <SelectTrigger className="w-28">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="7">最近 7 天</SelectItem>
            <SelectItem value="14">最近 14 天</SelectItem>
            <SelectItem value="30">最近 30 天</SelectItem>
            <SelectItem value="60">最近 60 天</SelectItem>
            <SelectItem value="90">最近 90 天</SelectItem>
          </SelectContent>
        </Select>
      </CardHeader>
      <CardContent>
        {error ? (
          <p className="text-destructive text-sm">{error}</p>
        ) : loading && !data ? (
          <p className="text-muted-foreground text-sm py-12 text-center">加载中...</p>
        ) : (
          <ResponsiveContainer width="100%" height={240}>
            <AreaChart data={series} margin={{ left: 0, right: 12, top: 8, bottom: 0 }}>
              <defs>
                <linearGradient id="domainTrendFill" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor="var(--chart-1)" stopOpacity={0.4} />
                  <stop offset="100%" stopColor="var(--chart-1)" stopOpacity={0.02} />
                </linearGradient>
              </defs>
              <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" />
              <XAxis dataKey="date" tick={{ fontSize: 11, fill: 'var(--muted-foreground)' }} />
              <YAxis tick={{ fontSize: 11, fill: 'var(--muted-foreground)' }} width={48} />
              <Tooltip
                contentStyle={{
                  background: 'var(--popover)',
                  border: '1px solid var(--border)',
                  borderRadius: 6,
                  color: 'var(--popover-foreground)',
                  fontSize: 12,
                }}
                labelStyle={{ color: 'var(--muted-foreground)' }}
                formatter={(value) => [Number(value).toLocaleString(), '访问']}
              />
              <Area
                type="monotone"
                dataKey="count"
                stroke="var(--chart-1)"
                fill="url(#domainTrendFill)"
                strokeWidth={2}
              />
            </AreaChart>
          </ResponsiveContainer>
        )}
      </CardContent>
    </Card>
  )
}
