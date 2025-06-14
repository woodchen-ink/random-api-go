'use client'

import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Switch } from '@/components/ui/switch'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import DataSourceManagement from './DataSourceManagement'
import type { APIEndpoint } from '@/types/admin'
import { authenticatedFetch } from '@/lib/auth'
import {
  DndContext,
  closestCenter,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
  DragEndEvent,
} from '@dnd-kit/core'
import {
  arrayMove,
  SortableContext,
  sortableKeyboardCoordinates,
  verticalListSortingStrategy,
} from '@dnd-kit/sortable'
import {
  useSortable,
} from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { GripVertical } from 'lucide-react'

interface EndpointsTabProps {
  endpoints: APIEndpoint[]
  onCreateEndpoint: (data: Partial<APIEndpoint>) => void
  onUpdateEndpoints: () => void
}

// 可拖拽的表格行组件
function SortableTableRow({ endpoint, onManageDataSources }: {
  endpoint: APIEndpoint
  onManageDataSources: (endpoint: APIEndpoint) => void
}) {
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id: endpoint.id })

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
  }

  return (
    <TableRow ref={setNodeRef} style={style} className={isDragging ? 'z-50' : ''}>
      <TableCell>
        <div
          {...attributes}
          {...listeners}
          className="flex items-center justify-center cursor-grab active:cursor-grabbing p-1 hover:bg-muted rounded"
        >
          <GripVertical className="h-4 w-4 text-muted-foreground" />
        </div>
      </TableCell>
      <TableCell className="font-medium">
        {endpoint.name}
      </TableCell>
      <TableCell>
        {endpoint.url}
      </TableCell>
      <TableCell>
        <span className={`inline-flex px-2 py-1 text-xs font-semibold rounded-full ${
          endpoint.is_active 
            ? 'bg-green-100 text-green-800 dark:bg-green-800 dark:text-green-100' 
            : 'bg-red-100 text-red-800 dark:bg-red-800 dark:text-red-100'
        }`}>
          {endpoint.is_active ? '启用' : '禁用'}
        </span>
      </TableCell>
      <TableCell>
        <span className={`inline-flex px-2 py-1 text-xs font-semibold rounded-full ${
          endpoint.show_on_homepage 
            ? 'bg-blue-100 text-blue-800 dark:bg-blue-800 dark:text-blue-100' 
            : 'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-100'
        }`}>
          {endpoint.show_on_homepage ? '显示' : '隐藏'}
        </span>
      </TableCell>
      <TableCell>
        {new Date(endpoint.created_at).toLocaleDateString()}
      </TableCell>
      <TableCell>
        <Button
          onClick={() => onManageDataSources(endpoint)}
          variant="outline"
          size="sm"
        >
          管理数据源
        </Button>
      </TableCell>
    </TableRow>
  )
}

export default function EndpointsTab({ endpoints, onCreateEndpoint, onUpdateEndpoints }: EndpointsTabProps) {
  const [showCreateForm, setShowCreateForm] = useState(false)
  const [selectedEndpoint, setSelectedEndpoint] = useState<APIEndpoint | null>(null)
  const [formData, setFormData] = useState({
    name: '',
    url: '',
    description: '',
    is_active: true,
    show_on_homepage: true
  })

  const sensors = useSensors(
    useSensor(PointerSensor),
    useSensor(KeyboardSensor, {
      coordinateGetter: sortableKeyboardCoordinates,
    })
  )

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    onCreateEndpoint(formData)
    setFormData({ name: '', url: '', description: '', is_active: true, show_on_homepage: true })
    setShowCreateForm(false)
  }

  const loadEndpointDataSources = async (endpointId: number) => {
    try {
      const response = await authenticatedFetch(`/api/admin/endpoints/${endpointId}/data-sources`)
      if (response.ok) {
        const data = await response.json()
        const endpoint = endpoints.find(e => e.id === endpointId)
        if (endpoint) {
          endpoint.data_sources = data.data || []
          setSelectedEndpoint({ ...endpoint })
        }
      }
    } catch (error) {
      console.error('Failed to load data sources:', error)
    }
  }

  const handleManageDataSources = (endpoint: APIEndpoint) => {
    setSelectedEndpoint(endpoint)
    loadEndpointDataSources(endpoint.id)
  }

  // 处理拖拽结束事件
  const handleDragEnd = async (event: DragEndEvent) => {
    const { active, over } = event

    if (!over || active.id === over.id) {
      return
    }

    const oldIndex = endpoints.findIndex(endpoint => endpoint.id === active.id)
    const newIndex = endpoints.findIndex(endpoint => endpoint.id === over.id)

    if (oldIndex === -1 || newIndex === -1) {
      return
    }

    // 创建新的排序数组
    const newEndpoints = arrayMove(endpoints, oldIndex, newIndex)

    // 更新排序值
    const endpointOrders = newEndpoints.map((endpoint, index) => ({
      id: endpoint.id,
      sort_order: index
    }))

    try {
      const response = await authenticatedFetch('/api/admin/endpoints/sort-order', {
        method: 'PUT',
        body: JSON.stringify({ endpoint_orders: endpointOrders }),
      })

      if (response.ok) {
        onUpdateEndpoints()
      } else {
        alert('更新排序失败')
      }
    } catch (error) {
      console.error('Failed to update sort order:', error)
      alert('更新排序失败')
    }
  }

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <h2 className="text-2xl font-bold tracking-tight">API端点管理</h2>
        <Button
          onClick={() => setShowCreateForm(true)}
        >
          创建端点
        </Button>
      </div>

      {showCreateForm && (
        <div className="bg-card rounded-lg border p-6 mb-6">
          <h3 className="text-lg font-medium mb-4">创建新端点</h3>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="name">端点名称</Label>
              <Input
                id="name"
                type="text"
                value={formData.name}
                onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                required
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="url">URL路径</Label>
              <Input
                id="url"
                type="text"
                value={formData.url}
                onChange={(e) => setFormData({ ...formData, url: e.target.value })}
                placeholder="例如: pic/anime"
                required
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="description">描述</Label>
              <Textarea
                id="description"
                value={formData.description}
                onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                rows={3}
              />
            </div>
            <div className="flex space-x-6">
              <div className="flex items-center space-x-2">
                <Switch
                  id="is_active"
                  checked={formData.is_active}
                  onCheckedChange={(checked) => setFormData({ ...formData, is_active: checked })}
                />
                <Label htmlFor="is_active">启用端点</Label>
              </div>
              <div className="flex items-center space-x-2">
                <Switch
                  id="show_on_homepage"
                  checked={formData.show_on_homepage}
                  onCheckedChange={(checked) => setFormData({ ...formData, show_on_homepage: checked })}
                />
                <Label htmlFor="show_on_homepage">显示在首页</Label>
              </div>
            </div>
            <div className="flex space-x-3">
              <Button type="submit">
                创建
              </Button>
              <Button
                type="button"
                onClick={() => setShowCreateForm(false)}
                variant="outline"
              >
                取消
              </Button>
            </div>
          </form>
        </div>
      )}

      <div className="rounded-md border">
        <DndContext
          sensors={sensors}
          collisionDetection={closestCenter}
          onDragEnd={handleDragEnd}
        >
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-16">拖拽</TableHead>
                <TableHead>名称</TableHead>
                <TableHead>URL</TableHead>
                <TableHead>状态</TableHead>
                <TableHead>首页显示</TableHead>
                <TableHead>创建时间</TableHead>
                <TableHead>操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <SortableContext
                items={endpoints.map(endpoint => endpoint.id)}
                strategy={verticalListSortingStrategy}
              >
                {endpoints.map((endpoint) => (
                  <SortableTableRow
                    key={endpoint.id}
                    endpoint={endpoint}
                    onManageDataSources={handleManageDataSources}
                  />
                ))}
              </SortableContext>
            </TableBody>
          </Table>
        </DndContext>
      </div>

      {/* 数据源管理弹窗 */}
      {selectedEndpoint && (
        <DataSourceManagement
          endpoint={selectedEndpoint}
          onClose={() => setSelectedEndpoint(null)}
          onUpdate={() => loadEndpointDataSources(selectedEndpoint.id)}
        />
      )}
    </div>
  )
} 