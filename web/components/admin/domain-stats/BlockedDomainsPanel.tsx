'use client'

import { useCallback, useEffect, useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { authenticatedFetch } from '@/lib/auth'
import type { BlockedDomain } from '@/types/admin'

// 黑名单管理: 显式增删, 与排行表里的开关共享一份数据
export default function BlockedDomainsPanel({ refreshKey }: { refreshKey: number }) {
  const [rows, setRows] = useState<BlockedDomain[] | null>(null)
  const [domain, setDomain] = useState('')
  const [reason, setReason] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    try {
      setError(null)
      const resp = await authenticatedFetch('/api/admin/blocked-domains')
      if (!resp.ok) throw new Error('请求失败')
      const json = await resp.json()
      setRows(json.data ?? [])
    } catch (e) {
      setError((e as Error).message)
    }
  }, [])

  useEffect(() => {
    load()
  }, [load, refreshKey])

  const add = async () => {
    const target = domain.trim().toLowerCase()
    if (!target) return
    try {
      setSubmitting(true)
      setError(null)
      const resp = await authenticatedFetch('/api/admin/domain-stats/block', {
        method: 'PUT',
        body: JSON.stringify({ domain: target, blocked: true, reason: reason.trim() }),
      })
      if (!resp.ok) throw new Error(await resp.text())
      setDomain('')
      setReason('')
      await load()
    } catch (e) {
      setError((e as Error).message)
    } finally {
      setSubmitting(false)
    }
  }

  const remove = async (target: string) => {
    try {
      const resp = await authenticatedFetch('/api/admin/domain-stats/block', {
        method: 'PUT',
        body: JSON.stringify({ domain: target, blocked: false }),
      })
      if (!resp.ok) throw new Error(await resp.text())
      await load()
    } catch (e) {
      setError((e as Error).message)
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">黑名单管理</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
          <Input
            placeholder="referer host, 例如 evil.example.com"
            value={domain}
            onChange={(e) => setDomain(e.target.value)}
            className="sm:max-w-xs"
          />
          <Input
            placeholder="备注 (可选)"
            value={reason}
            onChange={(e) => setReason(e.target.value)}
            className="sm:max-w-xs"
          />
          <Button onClick={add} disabled={submitting || !domain.trim()}>
            {submitting ? '提交中...' : '加入黑名单'}
          </Button>
        </div>

        {error && <p className="text-destructive text-sm">{error}</p>}

        {rows === null ? (
          <p className="text-muted-foreground text-sm py-6 text-center">加载中...</p>
        ) : rows.length === 0 ? (
          <p className="text-muted-foreground text-sm py-6 text-center">暂无禁用域名</p>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>域名</TableHead>
                <TableHead>备注</TableHead>
                <TableHead className="w-32 text-right">添加时间</TableHead>
                <TableHead className="w-20 text-right">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map(r => (
                <TableRow key={r.id}>
                  <TableCell className="font-mono">{r.domain}</TableCell>
                  <TableCell className="text-sm text-muted-foreground">{r.reason || '-'}</TableCell>
                  <TableCell className="text-right text-xs text-muted-foreground">
                    {new Date(r.created_at).toLocaleDateString()}
                  </TableCell>
                  <TableCell className="text-right">
                    <Button variant="ghost" size="sm" onClick={() => remove(r.domain)}>
                      解禁
                    </Button>
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
