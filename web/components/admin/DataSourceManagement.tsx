'use client'

import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

import { Switch } from '@/components/ui/switch'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import DataSourceConfigForm from './DataSourceConfigForm'
import type { APIEndpoint, DataSource } from '@/types/admin'
import { authenticatedFetch } from '@/lib/auth'

interface DataSourceManagementProps {
  endpoint: APIEndpoint
  onClose: () => void
  onUpdate: () => void
}

export default function DataSourceManagement({ 
  endpoint, 
  onClose, 
  onUpdate 
}: DataSourceManagementProps) {
  const [showCreateForm, setShowCreateForm] = useState(false)
  const [editingDataSource, setEditingDataSource] = useState<DataSource | null>(null)
  const [formData, setFormData] = useState({
    name: '',
    type: 'manual' as 'lankong' | 'manual' | 'api_get' | 'api_post' | 'endpoint',
    config: '',
    cache_duration: 3600,
    is_active: true
  })

  const createDataSource = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      // 处理配置数据
      let config = formData.config
      if (formData.type === 'manual') {
        // 将每行URL转换为JSON格式，过滤掉空行和注释行
        const urls = formData.config.split('\n')
          .map(url => url.trim())
          .filter(url => url.length > 0 && !url.startsWith('#'))
        config = JSON.stringify({ urls })
      }

      const response = await authenticatedFetch(`/api/admin/endpoints/${endpoint.id}/data-sources`, {
        method: 'POST',
        body: JSON.stringify({
          ...formData,
          config,
          endpoint_id: endpoint.id
        }),
      })

      if (response.ok) {
        onUpdate()
        setFormData({ name: '', type: 'manual' as const, config: '', cache_duration: 3600, is_active: true })
        setShowCreateForm(false)
        alert('数据源创建成功')
      } else {
        alert('创建数据源失败')
      }
    } catch (error) {
      console.error('Failed to create data source:', error)
      alert('创建数据源失败')
    }
  }

  const syncDataSource = async (dataSourceId: number) => {
    try {
      const response = await authenticatedFetch(`/api/admin/data-sources/${dataSourceId}/sync`, {
        method: 'POST',
      })

      if (response.ok) {
        onUpdate()
        alert('数据源同步成功')
      } else {
        alert('数据源同步失败')
      }
    } catch (error) {
      console.error('Failed to sync data source:', error)
      alert('数据源同步失败')
    }
  }

  const updateDataSource = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!editingDataSource) return

    try {
      // 处理配置数据
      let config = formData.config
      if (formData.type === 'manual') {
        // 将每行URL转换为JSON格式，过滤掉空行和注释行
        const urls = formData.config.split('\n')
          .map(url => url.trim())
          .filter(url => url.length > 0 && !url.startsWith('#'))
        config = JSON.stringify({ urls })
      }

      const response = await authenticatedFetch(`/api/admin/data-sources/${editingDataSource.id}`, {
        method: 'PUT',
        body: JSON.stringify({
          ...formData,
          config
        }),
      })

      if (response.ok) {
        onUpdate()
        setFormData({ name: '', type: 'manual' as const, config: '', cache_duration: 3600, is_active: true })
        setEditingDataSource(null)
        alert('数据源更新成功')
      } else {
        alert('更新数据源失败')
      }
    } catch (error) {
      console.error('Failed to update data source:', error)
      alert('更新数据源失败')
    }
  }

  const startEditDataSource = (dataSource: DataSource) => {
    setEditingDataSource(dataSource)
    
    // 处理配置数据回显
    let config = dataSource.config
    if (dataSource.type === 'manual') {
      try {
        // 将JSON格式转换为每行一个URL的格式
        const parsed = JSON.parse(dataSource.config)
        if (parsed.urls && Array.isArray(parsed.urls)) {
          config = parsed.urls.join('\n')
        }
      } catch (error) {
        console.error('Failed to parse manual config:', error)
        // 如果解析失败，保持原始配置
      }
    }
    
    setFormData({
      name: dataSource.name,
      type: dataSource.type,
      config: config,
      cache_duration: dataSource.cache_duration,
      is_active: dataSource.is_active
    })
    setShowCreateForm(false) // 关闭创建表单
  }

  const cancelEdit = () => {
    setEditingDataSource(null)
    setFormData({ name: '', type: 'manual' as const, config: '', cache_duration: 3600, is_active: true })
  }

  const deleteDataSource = async (dataSourceId: number) => {
    if (!confirm('确定要删除这个数据源吗？')) {
      return
    }

    try {
      const response = await authenticatedFetch(`/api/admin/data-sources/${dataSourceId}`, {
        method: 'DELETE',
      })

      if (response.ok) {
        onUpdate()
        alert('数据源删除成功')
      } else {
        alert('数据源删除失败')
      }
    } catch (error) {
      console.error('Failed to delete data source:', error)
      alert('数据源删除失败')
    }
  }



  const getTypeDisplayName = (type: string) => {
    switch (type) {
      case 'manual': return '手动'
      case 'lankong': return '兰空图床'
      case 'api_get': return 'GET接口'
      case 'api_post': return 'POST接口'
      case 'endpoint': return '已有端点'
      default: return type
    }
  }

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div className="bg-white dark:bg-gray-900 rounded-lg border shadow-xl max-w-6xl w-full max-h-[90vh] overflow-hidden flex flex-col">
        <div className="p-6 border-b border-gray-200 dark:border-gray-700 flex-shrink-0">
          <div className="flex justify-between items-center">
            <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
              管理数据源 - {endpoint.name}
            </h3>
            <Button
              onClick={onClose}
              variant="ghost"
              size="sm"
              className="text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
            >
              ✕
            </Button>
          </div>
        </div>

        <div className="p-6 flex-1 overflow-y-auto">
          <div className="flex justify-between items-center mb-4">
            <h4 className="text-md font-medium text-gray-900 dark:text-gray-100">数据源列表</h4>
            <Button
              onClick={() => {
                setShowCreateForm(true)
                setEditingDataSource(null)
                setFormData({ name: '', type: 'manual' as const, config: '', cache_duration: 3600, is_active: true })
              }}
              size="sm"
            >
              添加数据源
            </Button>
          </div>

          {(showCreateForm || editingDataSource) && (
            <div className="bg-gray-50 dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-4 mb-4">
              <h5 className="text-sm font-medium mb-3 text-gray-900 dark:text-gray-100">
                {editingDataSource ? '编辑数据源' : '创建新数据源'}
              </h5>
              <form onSubmit={editingDataSource ? updateDataSource : createDataSource} className="space-y-3">
                <div className="grid grid-cols-2 gap-3">
                  <div className="space-y-1">
                    <Label htmlFor="ds-name">数据源名称</Label>
                    <Input
                      id="ds-name"
                      type="text"
                      value={formData.name}
                      onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                      required
                    />
                  </div>
                  <div className="space-y-1">
                    <Label htmlFor="ds-type">数据源类型</Label>
                    <select
                      id="ds-type"
                      value={formData.type}
                      onChange={(e) => setFormData({ ...formData, type: e.target.value as 'lankong' | 'manual' | 'api_get' | 'api_post' | 'endpoint' })}
                      className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                    >
                      <option value="manual">手动数据链接</option>
                      <option value="lankong">兰空图床接口</option>
                      <option value="api_get">GET接口</option>
                      <option value="api_post">POST接口</option>
                      <option value="endpoint">已有端点</option>
                    </select>
                  </div>
                </div>
                <DataSourceConfigForm
                  type={formData.type}
                  config={formData.config}
                  onChange={(config) => setFormData({ ...formData, config })}
                />
                <div className="grid grid-cols-2 gap-3">
                  <div className="space-y-1">
                    <Label htmlFor="ds-cache">缓存时长(秒)</Label>
                    <Input
                      id="ds-cache"
                      type="number"
                      value={formData.cache_duration}
                      onChange={(e) => setFormData({ ...formData, cache_duration: parseInt(e.target.value) || 0 })}
                      min="0"
                    />
                    <p className="text-xs text-muted-foreground">
                      设置为0表示不缓存，建议设置3600秒(1小时)以上
                    </p>
                  </div>
                  <div className="flex items-center space-x-2 pt-6">
                    <Switch
                      id="ds-active"
                      checked={formData.is_active}
                      onCheckedChange={(checked) => setFormData({ ...formData, is_active: checked })}
                    />
                    <Label htmlFor="ds-active">启用数据源</Label>
                  </div>
                </div>
                <div className="flex space-x-2">
                  <Button type="submit" size="sm">
                    {editingDataSource ? '更新' : '创建'}
                  </Button>
                  <Button
                    type="button"
                    onClick={editingDataSource ? cancelEdit : () => setShowCreateForm(false)}
                    variant="outline"
                    size="sm"
                  >
                    取消
                  </Button>
                </div>
              </form>
            </div>
          )}

          <div className="rounded-md border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>名称</TableHead>
                  <TableHead>类型</TableHead>
                  <TableHead>状态</TableHead>
                  <TableHead>缓存时长</TableHead>
                  <TableHead>最后同步</TableHead>
                  <TableHead>操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {endpoint.data_sources && endpoint.data_sources.length > 0 ? (
                  endpoint.data_sources.map((dataSource) => (
                    <TableRow key={dataSource.id}>
                      <TableCell className="font-medium">
                        {dataSource.name}
                      </TableCell>
                      <TableCell>
                        <span className="inline-flex px-2 py-1 text-xs font-semibold rounded-full bg-blue-100 text-blue-800 dark:bg-blue-800 dark:text-blue-100">
                          {getTypeDisplayName(dataSource.type)}
                        </span>
                      </TableCell>
                      <TableCell>
                        <span className={`inline-flex px-2 py-1 text-xs font-semibold rounded-full ${
                          dataSource.is_active 
                            ? 'bg-green-100 text-green-800 dark:bg-green-800 dark:text-green-100' 
                            : 'bg-red-100 text-red-800 dark:bg-red-800 dark:text-red-100'
                        }`}>
                          {dataSource.is_active ? '启用' : '禁用'}
                        </span>
                      </TableCell>
                      <TableCell>
                        {dataSource.cache_duration > 0 ? `${dataSource.cache_duration}秒` : '不缓存'}
                      </TableCell>
                      <TableCell>
                        {dataSource.last_sync ? new Date(dataSource.last_sync).toLocaleString() : '未同步'}
                      </TableCell>
                      <TableCell>
                        <div className="flex space-x-1">
                          <Button 
                            variant="outline" 
                            size="sm"
                            onClick={() => startEditDataSource(dataSource)}
                          >
                            编辑
                          </Button>
                          <Button 
                            variant="outline" 
                            size="sm"
                            onClick={() => syncDataSource(dataSource.id)}
                          >
                            同步
                          </Button>
                          <Button 
                            variant="outline" 
                            size="sm"
                            onClick={() => deleteDataSource(dataSource.id)}
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
                    <TableCell colSpan={6} className="text-center text-muted-foreground py-8">
                      暂无数据源，点击&quot;添加数据源&quot;开始配置
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </div>
        </div>
      </div>
    </div>
  )
} 