'use client'

import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Switch } from '@/components/ui/switch'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog'
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
  onUpdateEndpoint: (id: number, data: Partial<APIEndpoint>) => void
  onUpdateEndpoints: () => void
}

// 可拖拽的表格行组件
function SortableTableRow({ endpoint, onManageDataSources, onEditEndpoint }: {
  endpoint: APIEndpoint
  onManageDataSources: (endpoint: APIEndpoint) => void
  onEditEndpoint: (endpoint: APIEndpoint) => void
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
        <a 
          href={`/${endpoint.url}`} 
          target="_blank" 
          rel="noopener noreferrer"
          className="text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300 underline hover:no-underline"
        >
          {endpoint.url}
        </a>
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
        <div className="flex space-x-2">
          <Button
            onClick={() => onEditEndpoint(endpoint)}
            variant="outline"
            size="sm"
          >
            编辑
          </Button>
          <Button
            onClick={() => onManageDataSources(endpoint)}
            variant="outline"
            size="sm"
          >
            管理数据源
          </Button>
        </div>
      </TableCell>
    </TableRow>
  )
}

export default function EndpointsTab({ endpoints, onCreateEndpoint, onUpdateEndpoint, onUpdateEndpoints }: EndpointsTabProps) {
  const [showCreateDialog, setShowCreateDialog] = useState(false)
  const [showEditDialog, setShowEditDialog] = useState(false)
  const [editingEndpoint, setEditingEndpoint] = useState<APIEndpoint | null>(null)
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
    setShowCreateDialog(false)
  }

  const handleEditSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (editingEndpoint) {
      onUpdateEndpoint(editingEndpoint.id, formData)
      setFormData({ name: '', url: '', description: '', is_active: true, show_on_homepage: true })
      setShowEditDialog(false)
      setEditingEndpoint(null)
    }
  }

  const handleEditEndpoint = (endpoint: APIEndpoint) => {
    setEditingEndpoint(endpoint)
    setFormData({
      name: endpoint.name,
      url: endpoint.url,
      description: endpoint.description,
      is_active: endpoint.is_active,
      show_on_homepage: endpoint.show_on_homepage
    })
    setShowEditDialog(true)
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
          onClick={() => setShowCreateDialog(true)}
        >
          创建端点
        </Button>
      </div>

      <Dialog open={showCreateDialog} onOpenChange={setShowCreateDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>创建新端点</DialogTitle>
          </DialogHeader>
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
            <DialogFooter>
              <Button type="submit">
                创建
              </Button>
              <Button
                type="button"
                onClick={() => setShowCreateDialog(false)}
                variant="outline"
              >
                取消
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <Dialog open={showEditDialog} onOpenChange={(open) => {
        if (!open) {
          setShowEditDialog(false)
          setEditingEndpoint(null)
          setFormData({ name: '', url: '', description: '', is_active: true, show_on_homepage: true })
        }
      }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>编辑端点</DialogTitle>
          </DialogHeader>
          <form onSubmit={handleEditSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="edit-name">端点名称</Label>
              <Input
                id="edit-name"
                type="text"
                value={formData.name}
                onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                required
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="edit-url">URL路径</Label>
              <Input
                id="edit-url"
                type="text"
                value={formData.url}
                onChange={(e) => setFormData({ ...formData, url: e.target.value })}
                placeholder="例如: pic/anime"
                required
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="edit-description">描述</Label>
              <Textarea
                id="edit-description"
                value={formData.description}
                onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                rows={3}
              />
            </div>
            <div className="flex space-x-6">
              <div className="flex items-center space-x-2">
                <Switch
                  id="edit-is_active"
                  checked={formData.is_active}
                  onCheckedChange={(checked) => setFormData({ ...formData, is_active: checked })}
                />
                <Label htmlFor="edit-is_active">启用端点</Label>
              </div>
              <div className="flex items-center space-x-2">
                <Switch
                  id="edit-show_on_homepage"
                  checked={formData.show_on_homepage}
                  onCheckedChange={(checked) => setFormData({ ...formData, show_on_homepage: checked })}
                />
                <Label htmlFor="edit-show_on_homepage">显示在首页</Label>
              </div>
            </div>
            <DialogFooter>
              <Button type="submit">
                更新
              </Button>
              <Button
                type="button"
                onClick={() => {
                  setShowEditDialog(false)
                  setEditingEndpoint(null)
                  setFormData({ name: '', url: '', description: '', is_active: true, show_on_homepage: true })
                }}
                variant="outline"
              >
                取消
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

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
                    onEditEndpoint={handleEditEndpoint}
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