'use client'

import { useMemo } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Bar, BarChart, CartesianGrid, Cell, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts'
import type { DomainStatsResult } from '@/types/admin'

const CHART_COLORS = ['var(--chart-1)', 'var(--chart-2)', 'var(--chart-3)', 'var(--chart-4)', 'var(--chart-5)']

// Top N 域名柱状图: 不含 direct/unknown, 用于一眼识别前几大调用方
export default function DomainTopBarChart({ data }: { data: DomainStatsResult[] | null }) {
  const series = useMemo(() => {
    if (!data) return []
    return data
      .filter(d => d.domain !== 'direct' && d.domain !== 'unknown')
      .slice(0, 8)
      .map(d => ({ domain: d.domain, count: d.count, blocked: d.is_blocked }))
  }, [data])

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Top 8 域名 (排除直接访问 / 未知)</CardTitle>
      </CardHeader>
      <CardContent>
        {series.length === 0 ? (
          <p className="text-muted-foreground text-sm py-8 text-center">暂无数据</p>
        ) : (
          <ResponsiveContainer width="100%" height={240}>
            <BarChart data={series} margin={{ left: 0, right: 12, top: 8, bottom: 40 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" vertical={false} />
              <XAxis
                dataKey="domain"
                tick={{ fontSize: 11, fill: 'var(--muted-foreground)' }}
                angle={-25}
                textAnchor="end"
                height={50}
                interval={0}
              />
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
              <Bar dataKey="count" radius={[6, 6, 0, 0]}>
                {series.map((entry, i) => (
                  <Cell
                    key={entry.domain}
                    fill={entry.blocked ? 'var(--destructive)' : CHART_COLORS[i % CHART_COLORS.length]}
                  />
                ))}
              </Bar>
            </BarChart>
          </ResponsiveContainer>
        )}
      </CardContent>
    </Card>
  )
}
