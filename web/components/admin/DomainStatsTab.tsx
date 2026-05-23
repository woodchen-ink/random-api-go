'use client'

import { useCallback, useEffect, useMemo, useState } from 'react'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Button } from '@/components/ui/button'
import { authenticatedFetch } from '@/lib/auth'
import { toast } from 'sonner'
import type { DomainStatsData, DomainStatsResult } from '@/types/admin'
import DomainTrendChart from './domain-stats/DomainTrendChart'
import DomainTopBarChart from './domain-stats/DomainTopBarChart'
import DomainRankTable from './domain-stats/DomainRankTable'
import DomainPathDrilldown from './domain-stats/DomainPathDrilldown'
import BlockedDomainsPanel from './domain-stats/BlockedDomainsPanel'

type DrillRange = '24h' | '7d' | '30d' | 'total'

// 域名统计页面: 拆为「总览」「24h」「7 天」「30 天」「总排行」「黑名单」六个 sub-tab
// 数据手动刷新, 不自动轮询, 避免点开下钻或切换禁用时表格抖动
export default function DomainStatsTab() {
  const [data, setData] = useState<DomainStatsData | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [lastUpdateTime, setLastUpdateTime] = useState<Date | null>(null)
  const [blocklistBump, setBlocklistBump] = useState(0)

  const [drillDomain, setDrillDomain] = useState<string | null>(null)
  const [drillRange, setDrillRange] = useState<DrillRange>('24h')

  const load = useCallback(async () => {
    try {
      setLoading(true)
      setError(null)
      const resp = await authenticatedFetch('/api/admin/domain-stats')
      if (!resp.ok) throw new Error('获取域名统计失败')
      const json = await resp.json()
      setData(json.data as DomainStatsData)
      setLastUpdateTime(new Date())
    } catch (e) {
      setError((e as Error).message)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    load()
  }, [load])

  // 切换某个域名的禁用状态; 成功后局部更新, 不重新拉整张表
  const toggleBlock = useCallback(async (domain: string, blocked: boolean) => {
    try {
      const resp = await authenticatedFetch('/api/admin/domain-stats/block', {
        method: 'PUT',
        body: JSON.stringify({ domain, blocked }),
      })
      if (!resp.ok) {
        const text = await resp.text()
        throw new Error(text || '操作失败')
      }
      setData(prev => {
        if (!prev) return prev
        const patch = (list: DomainStatsResult[]) =>
          list.map(r => (r.domain === domain ? { ...r, is_blocked: blocked } : r))
        return {
          top_24_hours: patch(prev.top_24_hours),
          top_7_days: patch(prev.top_7_days),
          top_30_days: patch(prev.top_30_days),
          top_total: patch(prev.top_total),
        }
      })
      setBlocklistBump(v => v + 1)
      toast.success(blocked ? `已禁用 ${domain}` : `已解除 ${domain}`)
    } catch (e) {
      toast.error((e as Error).message)
    }
  }, [])

  const pickDomain = useCallback((range: DrillRange) => (domain: string) => {
    if (domain === 'direct' || domain === 'unknown') return
    setDrillRange(range)
    setDrillDomain(domain)
  }, [])

  const top24 = data?.top_24_hours ?? null
  const top7 = data?.top_7_days ?? null
  const top30 = data?.top_30_days ?? null
  const totalRank = data?.top_total ?? null

  const summary = useMemo(() => {
    const sum = (list: DomainStatsResult[] | null) => (list ?? []).reduce((s, r) => s + r.count, 0)
    return {
      day: sum(top24),
      week: sum(top7),
      month: sum(top30),
      total: sum(totalRank),
    }
  }, [top24, top7, top30, totalRank])

  if (error && !data) {
    return (
      <div className="flex flex-col items-center justify-center py-12 space-y-4">
        <p className="text-destructive">{error}</p>
        <Button onClick={load}>重试</Button>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h2 className="text-2xl font-bold">域名统计</h2>
          {lastUpdateTime && (
            <p className="text-sm text-muted-foreground mt-1">
              最后更新: {lastUpdateTime.toLocaleTimeString()}
            </p>
          )}
        </div>
        <Button onClick={load} disabled={loading} variant="default">
          {loading ? '刷新中...' : '刷新'}
        </Button>
      </div>

      <Tabs defaultValue="overview" className="w-full">
        <TabsList>
          <TabsTrigger value="overview">总览</TabsTrigger>
          <TabsTrigger value="d1">24 小时</TabsTrigger>
          <TabsTrigger value="d7">7 天</TabsTrigger>
          <TabsTrigger value="d30">30 天</TabsTrigger>
          <TabsTrigger value="total">总排行</TabsTrigger>
          <TabsTrigger value="blocked">黑名单</TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="space-y-4 pt-2">
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
            <SummaryCard label="今日访问" value={summary.day} />
            <SummaryCard label="7 天访问" value={summary.week} />
            <SummaryCard label="30 天访问" value={summary.month} />
            <SummaryCard label="历史合计 (Top 30)" value={summary.total} />
          </div>
          <div className="grid gap-4 lg:grid-cols-2">
            <DomainTrendChart />
            <DomainTopBarChart data={top30} />
          </div>
        </TabsContent>

        <TabsContent value="d1" className="pt-2">
          <DomainRankTable
            title="24 小时内访问最多的域名 (Top 30)"
            data={top24}
            loading={loading}
            onPickDomain={pickDomain('24h')}
            onToggleBlock={toggleBlock}
          />
        </TabsContent>

        <TabsContent value="d7" className="pt-2">
          <DomainRankTable
            title="最近 7 天访问最多的域名 (Top 30)"
            data={top7}
            loading={loading}
            onPickDomain={pickDomain('7d')}
            onToggleBlock={toggleBlock}
          />
        </TabsContent>

        <TabsContent value="d30" className="pt-2">
          <DomainRankTable
            title="最近 30 天访问最多的域名 (Top 30)"
            data={top30}
            loading={loading}
            onPickDomain={pickDomain('30d')}
            onToggleBlock={toggleBlock}
          />
        </TabsContent>

        <TabsContent value="total" className="pt-2">
          <DomainRankTable
            title="历史总访问最多的域名 (Top 30)"
            data={totalRank}
            loading={loading}
            onPickDomain={pickDomain('total')}
            onToggleBlock={toggleBlock}
          />
        </TabsContent>

        <TabsContent value="blocked" className="pt-2">
          <BlockedDomainsPanel refreshKey={blocklistBump} />
        </TabsContent>
      </Tabs>

      <DomainPathDrilldown
        domain={drillDomain}
        open={!!drillDomain}
        onOpenChange={(v) => !v && setDrillDomain(null)}
        defaultRange={drillRange}
      />
    </div>
  )
}

function SummaryCard({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-lg border bg-card p-4">
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className="text-2xl font-semibold tabular-nums mt-1">{value.toLocaleString()}</p>
    </div>
  )
}
