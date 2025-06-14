'use client'

import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import type { URLReplaceRule, APIEndpoint } from '@/types/admin'

interface URLRulesTabProps {
  rules: URLReplaceRule[]
  endpoints: APIEndpoint[]
  onCreateRule?: (data: Partial<URLReplaceRule>) => void
  onUpdateRule?: (id: number, data: Partial<URLReplaceRule>) => void
  onDeleteRule?: (id: number) => void
}

export default function URLRulesTab({ 
  rules, 
  endpoints,
  onCreateRule, 
  onUpdateRule, 
  onDeleteRule 
}: URLRulesTabProps) {
  const [showCreateForm, setShowCreateForm] = useState(false)
  const [editingRule, setEditingRule] = useState<URLReplaceRule | null>(null)
  const [formData, setFormData] = useState({
    name: '',
    from_url: '',
    to_url: '',
    endpoint_id: undefined as number | undefined,
    is_active: true
  })

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (editingRule) {
      // 更新规则
      if (onUpdateRule) {
        onUpdateRule(editingRule.id, formData)
        setEditingRule(null)
      }
    } else {
      // 创建规则
      if (onCreateRule) {
        onCreateRule(formData)
      }
    }
    setFormData({ name: '', from_url: '', to_url: '', endpoint_id: undefined, is_active: true })
    setShowCreateForm(false)
  }

  const handleEdit = (rule: URLReplaceRule) => {
    setEditingRule(rule)
    setFormData({
      name: rule.name,
      from_url: rule.from_url,
      to_url: rule.to_url,
      endpoint_id: rule.endpoint_id,
      is_active: rule.is_active
    })
    setShowCreateForm(true)
  }

  const handleCancelEdit = () => {
    setEditingRule(null)
    setFormData({ name: '', from_url: '', to_url: '', endpoint_id: undefined, is_active: true })
    setShowCreateForm(false)
  }

  const handleDelete = (ruleId: number) => {
    if (confirm('确定要删除这个URL替换规则吗？')) {
      if (onDeleteRule) {
        onDeleteRule(ruleId)
      }
    }
  }

  const toggleRuleStatus = (rule: URLReplaceRule) => {
    if (onUpdateRule) {
      onUpdateRule(rule.id, { is_active: !rule.is_active })
    }
  }

  const getEndpointName = (endpointId?: number) => {
    if (!endpointId) return '全局规则'
    const endpoint = endpoints.find(ep => ep.id === endpointId)
    return endpoint ? endpoint.name : `端点 ${endpointId}`
  }

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <h2 className="text-2xl font-bold tracking-tight">URL替换规则</h2>
        <Button onClick={() => setShowCreateForm(true)}>
          创建规则
        </Button>
      </div>

      {showCreateForm && (
        <div className="bg-card rounded-lg border p-6 mb-6">
          <h3 className="text-lg font-medium mb-4">
            {editingRule ? '编辑规则' : '创建新规则'}
          </h3>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="rule-name">规则名称</Label>
              <Input
                id="rule-name"
                type="text"
                value={formData.name}
                onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                placeholder="例如: 替换图床域名"
                required
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="endpoint-select">应用端点</Label>
              <Select
                value={formData.endpoint_id?.toString() || 'global'}
                onValueChange={(value) => setFormData({ 
                  ...formData, 
                  endpoint_id: value === 'global' ? undefined : parseInt(value)
                })}
              >
                <SelectTrigger>
                  <SelectValue placeholder="选择端点或设为全局规则" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="global">全局规则（应用于所有端点）</SelectItem>
                  {endpoints.map((endpoint) => (
                    <SelectItem key={endpoint.id} value={endpoint.id.toString()}>
                      {endpoint.name} ({endpoint.url})
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <p className="text-xs text-muted-foreground">
                选择特定端点或设为全局规则应用于所有端点
              </p>
            </div>
            
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="from-url">源URL模式</Label>
                <Input
                  id="from-url"
                  type="text"
                  value={formData.from_url}
                  onChange={(e) => setFormData({ ...formData, from_url: e.target.value })}
                  placeholder="例如: a.com"
                  required
                />
                <p className="text-xs text-muted-foreground">
                  支持域名或URL片段匹配
                </p>
              </div>
              <div className="space-y-2">
                <Label htmlFor="to-url">目标URL模式</Label>
                <Input
                  id="to-url"
                  type="text"
                  value={formData.to_url}
                  onChange={(e) => setFormData({ ...formData, to_url: e.target.value })}
                  placeholder="例如: b.com"
                  required
                />
                <p className="text-xs text-muted-foreground">
                  替换后的域名或URL片段
                </p>
              </div>
            </div>
            <div className="flex items-center space-x-2">
              <Switch
                id="rule-active"
                checked={formData.is_active}
                onCheckedChange={(checked) => setFormData({ ...formData, is_active: checked })}
              />
              <Label htmlFor="rule-active">启用规则</Label>
            </div>
            <div className="flex space-x-3">
              <Button type="submit">
                {editingRule ? '更新' : '创建'}
              </Button>
              <Button
                type="button"
                onClick={handleCancelEdit}
                variant="outline"
              >
                取消
              </Button>
            </div>
          </form>
        </div>
      )}

      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>规则名称</TableHead>
              <TableHead>应用端点</TableHead>
              <TableHead>源URL</TableHead>
              <TableHead>目标URL</TableHead>
              <TableHead>状态</TableHead>
              <TableHead>创建时间</TableHead>
              <TableHead>操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {rules.length > 0 ? (
              rules.map((rule) => (
                <TableRow key={rule.id}>
                  <TableCell className="font-medium">
                    {rule.name}
                  </TableCell>
                  <TableCell>
                    <span className={`inline-flex px-2 py-1 text-xs font-semibold rounded-full ${
                      rule.endpoint_id 
                        ? 'bg-blue-100 text-blue-800 dark:bg-blue-800 dark:text-blue-100' 
                        : 'bg-purple-100 text-purple-800 dark:bg-purple-800 dark:text-purple-100'
                    }`}>
                      {getEndpointName(rule.endpoint_id)}
                    </span>
                  </TableCell>
                  <TableCell>
                    <code className="bg-muted px-2 py-1 rounded text-sm">
                      {rule.from_url}
                    </code>
                  </TableCell>
                  <TableCell>
                    <code className="bg-muted px-2 py-1 rounded text-sm">
                      {rule.to_url}
                    </code>
                  </TableCell>
                  <TableCell>
                    <span className={`inline-flex px-2 py-1 text-xs font-semibold rounded-full ${
                      rule.is_active 
                        ? 'bg-green-100 text-green-800 dark:bg-green-800 dark:text-green-100' 
                        : 'bg-red-100 text-red-800 dark:bg-red-800 dark:text-red-100'
                    }`}>
                      {rule.is_active ? '启用' : '禁用'}
                    </span>
                  </TableCell>
                  <TableCell>
                    {new Date(rule.created_at).toLocaleDateString()}
                  </TableCell>
                  <TableCell>
                    <div className="flex space-x-1">
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => handleEdit(rule)}
                      >
                        编辑
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => toggleRuleStatus(rule)}
                      >
                        {rule.is_active ? '禁用' : '启用'}
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => handleDelete(rule.id)}
                        className="text-red-600 hover:text-red-700"
                      >
                        删除
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))
            ) : (
              <TableRow>
                <TableCell colSpan={7} className="text-center text-muted-foreground py-8">
                  暂无URL替换规则，点击&quot;创建规则&quot;开始配置
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>
    </div>
  )
} 