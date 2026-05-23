'use client'

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Switch } from '@/components/ui/switch'
import type { DomainStatsResult } from '@/types/admin'

// 域名排行表: 点击行 → 触发下钻; direct/unknown 不可禁用 (开关置灰)
export default function DomainRankTable({
  title,
  data,
  loading,
  onPickDomain,
  onToggleBlock,
}: {
  title: string
  data: DomainStatsResult[] | null
  loading: boolean
  onPickDomain: (domain: string) => void
  onToggleBlock: (domain: string, blocked: boolean) => Promise<void>
}) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">{title}</CardTitle>
      </CardHeader>
      <CardContent>
        {loading && !data ? (
          <p className="text-muted-foreground text-sm py-8 text-center">加载中...</p>
        ) : !data || data.length === 0 ? (
          <p className="text-muted-foreground text-sm py-8 text-center">暂无数据</p>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-12">#</TableHead>
                <TableHead>域名</TableHead>
                <TableHead className="text-right">访问次数</TableHead>
                <TableHead className="text-center w-20">禁用</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {data.map((item, index) => {
                const isSpecial = item.domain === 'direct' || item.domain === 'unknown'
                return (
                  <TableRow
                    key={item.domain}
                    className="cursor-pointer"
                    onClick={() => onPickDomain(item.domain)}
                  >
                    <TableCell className="font-medium">{index + 1}</TableCell>
                    <TableCell>
                      <span className="font-mono">
                        {item.domain === 'direct' ? (
                          <span className="text-muted-foreground">直接访问</span>
                        ) : item.domain === 'unknown' ? (
                          <span className="text-muted-foreground">未知来源</span>
                        ) : (
                          <a
                            href={`https://${item.domain}`}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="underline underline-offset-4 hover:text-accent transition-colors"
                            onClick={(e) => e.stopPropagation()}
                          >
                            {item.domain}
                          </a>
                        )}
                        {item.is_blocked && (
                          <span className="ml-2 text-xs text-destructive">已禁用</span>
                        )}
                      </span>
                    </TableCell>
                    <TableCell className="text-right tabular-nums">
                      {item.count.toLocaleString()}
                    </TableCell>
                    <TableCell className="text-center" onClick={(e) => e.stopPropagation()}>
                      <Switch
                        checked={item.is_blocked}
                        disabled={isSpecial}
                        onCheckedChange={(v) => onToggleBlock(item.domain, v)}
                        aria-label="禁用此域名"
                      />
                    </TableCell>
                  </TableRow>
                )
              })}
            </TableBody>
          </Table>
        )}
      </CardContent>
    </Card>
  )
}
