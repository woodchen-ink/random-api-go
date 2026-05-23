'use client'

import { useEffect, useState } from 'react'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { authenticatedFetch } from '@/lib/auth'
import type { DomainPathStat, DomainPathStatsResponse } from '@/types/admin'

type RangeKey = '24h' | '7d' | '30d' | 'total'

// 点击域名行后弹出, 展示该域名调用的所有 path 排行
// 默认进入时锁定到外层选中的 range, 弹窗内可临时切换不同时间范围
export default function DomainPathDrilldown({
  domain,
  open,
  onOpenChange,
  defaultRange,
}: {
  domain: string | null
  open: boolean
  onOpenChange: (v: boolean) => void
  defaultRange: RangeKey
}) {
  const [range, setRange] = useState<RangeKey>(defaultRange)
  const [paths, setPaths] = useState<DomainPathStat[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (open) setRange(defaultRange)
  }, [open, defaultRange])

  useEffect(() => {
    if (!open || !domain) return
    let cancelled = false
    const run = async () => {
      try {
        setLoading(true)
        setError(null)
        const resp = await authenticatedFetch(`/api/admin/domain-stats/paths?domain=${encodeURIComponent(domain)}&range=${range}`)
        if (!resp.ok) throw new Error('请求失败')
        const json = await resp.json()
        const data = json.data as DomainPathStatsResponse
        if (!cancelled) setPaths(data.paths ?? [])
      } catch (e) {
        if (!cancelled) setError((e as Error).message)
      } finally {
        if (!cancelled) setLoading(false)
      }
    }
    run()
    return () => {
      cancelled = true
    }
  }, [open, domain, range])

  const total = paths.reduce((s, p) => s + p.count, 0)

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle className="font-mono text-base">{domain ?? ''}</DialogTitle>
          <DialogDescription>
            该域名在所选时间范围内调用的端点路径排行 (合计 {total.toLocaleString()} 次)
          </DialogDescription>
        </DialogHeader>

        <div className="flex items-center justify-end pb-2">
          <Select value={range} onValueChange={(v) => setRange(v as RangeKey)}>
            <SelectTrigger className="w-32">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="24h">24 小时</SelectItem>
              <SelectItem value="7d">7 天</SelectItem>
              <SelectItem value="30d">30 天</SelectItem>
              <SelectItem value="total">总计</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div className="max-h-[420px] overflow-auto rounded-md border">
          {error ? (
            <p className="text-destructive text-sm p-4">{error}</p>
          ) : loading ? (
            <p className="text-muted-foreground text-sm p-4 text-center">加载中...</p>
          ) : paths.length === 0 ? (
            <p className="text-muted-foreground text-sm p-4 text-center">暂无数据</p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-12">#</TableHead>
                  <TableHead>路径</TableHead>
                  <TableHead className="text-right w-32">访问</TableHead>
                  <TableHead className="text-right w-20">占比</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {paths.map((p, i) => (
                  <TableRow key={p.path}>
                    <TableCell className="font-medium">{i + 1}</TableCell>
                    <TableCell className="font-mono text-sm">{p.path}</TableCell>
                    <TableCell className="text-right">{p.count.toLocaleString()}</TableCell>
                    <TableCell className="text-right text-muted-foreground">
                      {total === 0 ? '-' : `${((p.count / total) * 100).toFixed(1)}%`}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}
