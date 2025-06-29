'use client'

import { useState, useEffect } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Checkbox } from '@/components/ui/checkbox'
import { Trash2, Plus } from 'lucide-react'
import { authenticatedFetch } from '@/lib/auth'

interface DataSourceConfigFormProps {
  type: 'lankong' | 'manual' | 'api_get' | 'api_post' | 'endpoint' | 's3'
  config: string
  onChange: (config: string) => void
}

interface LankongConfig {
  api_token: string
  album_ids: string[]
  base_url?: string
}

interface APIConfig {
  url: string
  method?: string
  headers: { [key: string]: string }
  body?: string
  url_field: string
}

interface SavedToken {
  id: string
  name: string
  token: string
}

interface EndpointConfig {
  endpoint_ids: number[]
}

interface S3Config {
  endpoint: string
  bucket_name: string
  region: string
  access_key_id: string
  secret_access_key: string
  list_objects_version: string
  use_path_style: boolean
  remove_bucket: boolean
  custom_domain: string
  folder_path: string
  include_subfolders: boolean
  file_extensions: string[]
}

export default function DataSourceConfigForm({ type, config, onChange }: DataSourceConfigFormProps) {
  const [lankongConfig, setLankongConfig] = useState<LankongConfig>({
    api_token: '',
    album_ids: [''],
    base_url: ''
  })
  
  const [apiConfig, setAPIConfig] = useState<APIConfig>({
    url: '',
    method: type === 'api_post' ? 'POST' : 'GET',
    headers: {},
    body: '',
    url_field: 'url'
  })

  const [endpointConfig, setEndpointConfig] = useState<EndpointConfig>({
    endpoint_ids: []
  })

  const [s3Config, setS3Config] = useState<S3Config>({
    endpoint: '',
    bucket_name: '',
    region: '',
    access_key_id: '',
    secret_access_key: '',
    list_objects_version: 'v2',
    use_path_style: false,
    remove_bucket: false,
    custom_domain: '',
    folder_path: '',
    include_subfolders: true,
    file_extensions: []
  })

  const [availableEndpoints, setAvailableEndpoints] = useState<Array<{id: number, name: string, url: string}>>([])

  const [headerPairs, setHeaderPairs] = useState<Array<{key: string, value: string}>>([{key: '', value: ''}])
  const [extensionInputs, setExtensionInputs] = useState<string[]>([''])
  const [savedTokens, setSavedTokens] = useState<SavedToken[]>([])

  const [newTokenName, setNewTokenName] = useState<string>('')

  // 从localStorage加载保存的token
  useEffect(() => {
    const saved = localStorage.getItem('lankong_tokens')
    if (saved) {
      try {
        setSavedTokens(JSON.parse(saved))
      } catch (error) {
        console.error('Failed to parse saved tokens:', error)
      }
    }
  }, [])

  // 获取可用端点列表
  useEffect(() => {
    if (type === 'endpoint') {
      loadAvailableEndpoints()
    }
  }, [type])

  const loadAvailableEndpoints = async () => {
    try {
      const response = await authenticatedFetch('/api/admin/endpoints')
      if (response.ok) {
        const data = await response.json()
        setAvailableEndpoints(data.data || [])
      }
    } catch (error) {
      console.error('Failed to load endpoints:', error)
    }
  }

  // 解析现有配置
  useEffect(() => {
    if (!config) return

    try {
      const parsed = JSON.parse(config)
      
      if (type === 'lankong') {
        setLankongConfig({
          api_token: parsed.api_token || '',
          album_ids: parsed.album_ids || [''],
          base_url: parsed.base_url || ''
        })
      } else if (type === 'api_get' || type === 'api_post') {
        setAPIConfig({
          url: parsed.url || '',
          method: parsed.method || (type === 'api_post' ? 'POST' : 'GET'),
          headers: parsed.headers || {},
          body: parsed.body || '',
          url_field: parsed.url_field || 'url'
        })
        
        // 转换headers为键值对数组
        const pairs = Object.entries(parsed.headers || {}).map(([key, value]) => ({key, value: value as string}))
        if (pairs.length === 0) pairs.push({key: '', value: ''})
        setHeaderPairs(pairs)
      } else if (type === 'endpoint') {
        setEndpointConfig({
          endpoint_ids: parsed.endpoint_ids || []
        })
      } else if (type === 's3') {
        setS3Config({
          endpoint: parsed.endpoint || '',
          bucket_name: parsed.bucket_name || '',
          region: parsed.region || '',
          access_key_id: parsed.access_key_id || '',
          secret_access_key: parsed.secret_access_key || '',
          list_objects_version: parsed.list_objects_version || 'v2',
          use_path_style: parsed.use_path_style || false,
          remove_bucket: parsed.remove_bucket || false,
          custom_domain: parsed.custom_domain || '',
          folder_path: parsed.folder_path || '',
          include_subfolders: parsed.include_subfolders !== false,
          file_extensions: parsed.file_extensions || []
        })
        
        // 设置文件扩展名输入框
        const extensions = parsed.file_extensions || ['']
        if (extensions.length === 0) extensions.push('')
        setExtensionInputs(extensions)
      }
    } catch (error) {
      console.error('Failed to parse config:', error)
    }
  }, [config, type])

  // 保存token到localStorage
  const saveToken = () => {
    if (!newTokenName.trim() || !lankongConfig.api_token.trim()) {
      alert('请输入token名称和token值')
      return
    }

    const newToken: SavedToken = {
      id: Date.now().toString(),
      name: newTokenName.trim(),
      token: lankongConfig.api_token
    }

    const updated = [...savedTokens, newToken]
    setSavedTokens(updated)
    localStorage.setItem('lankong_tokens', JSON.stringify(updated))
    setNewTokenName('')
    alert('Token保存成功')
  }

  // 删除保存的token
  const deleteToken = (tokenId: string) => {
    if (!confirm('确定要删除这个token吗？')) return
    
    const updated = savedTokens.filter(t => t.id !== tokenId)
    setSavedTokens(updated)
    localStorage.setItem('lankong_tokens', JSON.stringify(updated))
  }

  // 更新兰空图床配置
  const updateConfig = (newConfig: LankongConfig | APIConfig) => {
    onChange(JSON.stringify(newConfig))
  }

  // 添加相册ID
  const addAlbumId = () => {
    const newConfig = {
      ...lankongConfig,
      album_ids: [...lankongConfig.album_ids, '']
    }
    setLankongConfig(newConfig)
    updateConfig(newConfig)
  }

  // 删除相册ID
  const removeAlbumId = (index: number) => {
    const newConfig = {
      ...lankongConfig,
      album_ids: lankongConfig.album_ids.filter((_, i) => i !== index)
    }
    setLankongConfig(newConfig)
    updateConfig(newConfig)
  }

  // 更新相册ID
  const updateAlbumId = (index: number, value: string) => {
    const newConfig = {
      ...lankongConfig,
      album_ids: lankongConfig.album_ids.map((id, i) => i === index ? value : id)
    }
    setLankongConfig(newConfig)
    updateConfig(newConfig)
  }

  // 添加请求头
  const addHeader = () => {
    setHeaderPairs([...headerPairs, {key: '', value: ''}])
  }

  // 删除请求头
  const removeHeader = (index: number) => {
    const newPairs = headerPairs.filter((_, i) => i !== index)
    setHeaderPairs(newPairs)
    updateAPIHeaders(newPairs)
  }

  // 更新请求头
  const updateHeader = (index: number, field: 'key' | 'value', value: string) => {
    const newPairs = headerPairs.map((pair, i) => 
      i === index ? { ...pair, [field]: value } : pair
    )
    setHeaderPairs(newPairs)
    updateAPIHeaders(newPairs)
  }

  // 更新API配置的headers
  const updateAPIHeaders = (pairs: Array<{key: string, value: string}>) => {
    const headers: { [key: string]: string } = {}
    pairs.forEach(pair => {
      if (pair.key.trim() && pair.value.trim()) {
        headers[pair.key.trim()] = pair.value.trim()
      }
    })
    
    const newConfig = { ...apiConfig, headers }
    setAPIConfig(newConfig)
    updateConfig(newConfig)
  }

  // 更新API配置
  const updateAPIConfig = (field: keyof APIConfig, value: string) => {
    // 对URL字段进行trim处理，去除前后空格
    const trimmedValue = field === 'url' ? value.trim() : value
    const newConfig = { ...apiConfig, [field]: trimmedValue }
    setAPIConfig(newConfig)
    updateConfig(newConfig)
  }

  // 更新端点配置
  const updateEndpointConfig = (endpointIds: number[]) => {
    const newConfig = { endpoint_ids: endpointIds }
    setEndpointConfig(newConfig)
    onChange(JSON.stringify(newConfig))
  }

  // 切换端点选择
  const toggleEndpoint = (endpointId: number) => {
    const currentIds = endpointConfig.endpoint_ids
    const newIds = currentIds.includes(endpointId)
      ? currentIds.filter(id => id !== endpointId)
      : [...currentIds, endpointId]
    updateEndpointConfig(newIds)
  }

  // 更新S3配置
  const updateS3Config = (field: keyof S3Config, value: string | boolean | string[]) => {
    const newConfig = { ...s3Config, [field]: value }
    setS3Config(newConfig)
    onChange(JSON.stringify(newConfig))
  }

  // 添加文件扩展名
  const addExtension = () => {
    const newExtensions = [...extensionInputs, '']
    setExtensionInputs(newExtensions)
  }

  // 删除文件扩展名
  const removeExtension = (index: number) => {
    const newExtensions = extensionInputs.filter((_, i) => i !== index)
    setExtensionInputs(newExtensions)
    updateS3Config('file_extensions', newExtensions.filter(ext => ext.trim() !== ''))
  }

  // 更新文件扩展名
  const updateExtension = (index: number, value: string) => {
    const newExtensions = extensionInputs.map((ext, i) => i === index ? value : ext)
    setExtensionInputs(newExtensions)
    updateS3Config('file_extensions', newExtensions.filter(ext => ext.trim() !== ''))
  }

  if (type === 'manual') {
    return (
      <div className="space-y-2">
        <Label htmlFor="manual-config">URL列表</Label>
        <Textarea
          id="manual-config"
          value={config}
          onChange={(e) => onChange(e.target.value)}
          placeholder="每行输入一个URL地址"
          rows={4}
        />
        <p className="text-xs text-muted-foreground">
          每行输入一个URL地址，以#开头的行将被视为注释
        </p>
      </div>
    )
  }

  if (type === 'lankong') {
    return (
      <div className="space-y-4">
        {/* Token管理 */}
        <Card>
          <CardHeader>
            <CardTitle className="text-sm">Token管理</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {/* 保存的Token列表 */}
            {savedTokens.length > 0 && (
              <div className="space-y-2">
                <Label className="text-xs">使用保存的Token</Label>
                <div className="space-y-1">
                  {savedTokens.map((token) => (
                    <div key={token.id} className="flex items-center gap-2 p-2 border rounded">
                      <span className="flex-1 text-sm">{token.name}</span>
                                             <Button
                         type="button"
                         size="sm"
                         variant="outline"
                         onClick={() => {
                          const newConfig = { ...lankongConfig, api_token: token.token }
                          setLankongConfig(newConfig)
                          updateConfig(newConfig)
                         }}
                       >
                         使用
                       </Button>
                      <Button
                        type="button"
                        size="sm"
                        variant="outline"
                        onClick={() => deleteToken(token.id)}
                        className="text-red-600 hover:text-red-700"
                      >
                        <Trash2 className="h-3 w-3" />
                      </Button>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* API Token */}
            <div className="space-y-2">
              <Label htmlFor="api-token">API Token</Label>
              <Input
                id="api-token"
                type="password"
                value={lankongConfig.api_token}
                onChange={(e) => {
                  const newConfig = { ...lankongConfig, api_token: e.target.value }
                  setLankongConfig(newConfig)
                  updateConfig(newConfig)
                }}
                placeholder="输入兰空图床API Token"
              />
            </div>

            {/* 保存Token */}
            <div className="flex gap-2">
              <Input
                placeholder="Token名称（如：主账号、备用账号）"
                value={newTokenName}
                onChange={(e) => setNewTokenName(e.target.value)}
                className="flex-1"
              />
              <Button
                type="button"
                size="sm"
                onClick={saveToken}
                disabled={!newTokenName.trim() || !lankongConfig.api_token.trim()}
              >
                保存Token
              </Button>
            </div>
          </CardContent>
        </Card>

        {/* 相册配置 */}
        <Card>
          <CardHeader>
            <CardTitle className="text-sm">相册配置</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {/* 相册ID列表 */}
            <div className="space-y-2">
              <Label>相册ID列表</Label>
              {lankongConfig.album_ids.map((albumId, index) => (
                <div key={index} className="flex gap-2">
                  <Input
                    value={albumId}
                    onChange={(e) => updateAlbumId(index, e.target.value)}
                    placeholder="输入相册ID"
                    className="flex-1"
                  />
                  {lankongConfig.album_ids.length > 1 && (
                    <Button
                      type="button"
                      size="sm"
                      variant="outline"
                      onClick={() => removeAlbumId(index)}
                      className="text-red-600 hover:text-red-700"
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  )}
                </div>
              ))}
              <Button
                type="button"
                size="sm"
                variant="outline"
                onClick={addAlbumId}
                className="w-full"
              >
                <Plus className="h-4 w-4 mr-1" />
                添加相册
              </Button>
            </div>

            {/* Base URL */}
            <div className="space-y-2">
              <Label htmlFor="base-url">Base URL（可选）</Label>
              <Input
                id="base-url"
                value={lankongConfig.base_url}
                onChange={(e) => {
                  const newConfig = { ...lankongConfig, base_url: e.target.value.trim() }
                  setLankongConfig(newConfig)
                  updateConfig(newConfig)
                }}
                placeholder="默认: https://img.czl.net/api/v1/images"
              />
              <p className="text-xs text-muted-foreground">
                留空使用默认地址
              </p>
            </div>
          </CardContent>
        </Card>
      </div>
    )
  }

  if (type === 'api_get' || type === 'api_post') {
    return (
      <div className="space-y-4">
        <Card>
          <CardHeader>
            <CardTitle className="text-sm">API配置</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {/* API URL */}
            <div className="space-y-2">
              <Label htmlFor="api-url">API地址</Label>
              <Input
                id="api-url"
                value={apiConfig.url}
                onChange={(e) => updateAPIConfig('url', e.target.value)}
                placeholder="https://api.example.com/images"
              />
            </div>

            {/* 请求头 */}
            <div className="space-y-2">
              <Label>请求头</Label>
              {headerPairs.map((pair, index) => (
                <div key={index} className="flex gap-2">
                  <Input
                    value={pair.key}
                    onChange={(e) => updateHeader(index, 'key', e.target.value)}
                    placeholder="Header名称"
                    className="flex-1"
                  />
                  <Input
                    value={pair.value}
                    onChange={(e) => updateHeader(index, 'value', e.target.value)}
                    placeholder="Header值"
                    className="flex-1"
                  />
                  {headerPairs.length > 1 && (
                    <Button
                      type="button"
                      size="sm"
                      variant="outline"
                      onClick={() => removeHeader(index)}
                      className="text-red-600 hover:text-red-700"
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  )}
                </div>
              ))}
              <Button
                type="button"
                size="sm"
                variant="outline"
                onClick={addHeader}
                className="w-full"
              >
                <Plus className="h-4 w-4 mr-1" />
                添加请求头
              </Button>
            </div>

            {/* POST请求体 */}
            {type === 'api_post' && (
              <div className="space-y-2">
                <Label htmlFor="request-body">请求体（JSON）</Label>
                <Textarea
                  id="request-body"
                  value={apiConfig.body}
                  onChange={(e) => updateAPIConfig('body', e.target.value)}
                  placeholder='{"key": "value"}'
                  rows={3}
                />
              </div>
            )}

            {/* URL字段路径 */}
            <div className="space-y-2">
              <Label htmlFor="url-field">URL字段路径</Label>
              <Input
                id="url-field"
                value={apiConfig.url_field}
                onChange={(e) => updateAPIConfig('url_field', e.target.value)}
                placeholder="data.url 或 urls.0 或 url"
              />
              <p className="text-xs text-muted-foreground">
                指定响应JSON中URL字段的路径，支持嵌套路径如 data.url 或数组索引如 urls.0
              </p>
            </div>
          </CardContent>
        </Card>
      </div>
    )
  }

  if (type === 'endpoint') {
    return (
      <div className="space-y-4">
        <Card>
          <CardHeader>
            <CardTitle className="text-sm">端点选择</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="space-y-2">
              <Label>选择要跳转的端点（可多选）</Label>
              {availableEndpoints.length === 0 ? (
                <p className="text-sm text-muted-foreground">正在加载端点列表...</p>
              ) : (
                <div className="space-y-2 max-h-60 overflow-y-auto">
                  {availableEndpoints.map((endpoint) => (
                    <div key={endpoint.id} className="flex items-center space-x-2">
                      <Checkbox
                        id={`endpoint-${endpoint.id}`}
                        checked={endpointConfig.endpoint_ids.includes(endpoint.id)}
                        onCheckedChange={() => toggleEndpoint(endpoint.id)}
                      />
                      <Label 
                        htmlFor={`endpoint-${endpoint.id}`}
                        className="flex-1 cursor-pointer"
                      >
                        <div className="flex flex-col">
                          <span className="font-medium">{endpoint.name}</span>
                          <span className="text-xs text-muted-foreground">/{endpoint.url}</span>
                        </div>
                      </Label>
                    </div>
                  ))}
                </div>
              )}
              {endpointConfig.endpoint_ids.length > 0 && (
                <p className="text-xs text-muted-foreground">
                  已选择 {endpointConfig.endpoint_ids.length} 个端点
                </p>
              )}
            </div>
          </CardContent>
        </Card>
      </div>
    )
  }

  if (type === 's3') {
    return (
      <div className="space-y-4">
        {/* 基础配置 */}
        <Card>
          <CardHeader>
            <CardTitle className="text-sm">基础配置</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-2">
                <Label htmlFor="s3-endpoint">S3端点地址</Label>
                <Input
                  id="s3-endpoint"
                  value={s3Config.endpoint}
                  onChange={(e) => updateS3Config('endpoint', e.target.value)}
                  placeholder="https://s3.amazonaws.com"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="s3-bucket">存储桶名称</Label>
                <Input
                  id="s3-bucket"
                  value={s3Config.bucket_name}
                  onChange={(e) => updateS3Config('bucket_name', e.target.value)}
                  placeholder="my-bucket"
                />
              </div>
            </div>
            
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-2">
                <Label htmlFor="s3-region">地区</Label>
                <Input
                  id="s3-region"
                  value={s3Config.region}
                  onChange={(e) => updateS3Config('region', e.target.value)}
                  placeholder="us-east-1"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="s3-version">列出对象版本</Label>
                <select
                  id="s3-version"
                  value={s3Config.list_objects_version}
                  onChange={(e) => updateS3Config('list_objects_version', e.target.value)}
                  className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                >
                  <option value="v2">v2</option>
                  <option value="v1">v1</option>
                </select>
              </div>
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-2">
                <Label htmlFor="s3-access-key">访问密钥ID</Label>
                <Input
                  id="s3-access-key"
                  value={s3Config.access_key_id}
                  onChange={(e) => updateS3Config('access_key_id', e.target.value)}
                  placeholder="AKIAIOSFODNN7EXAMPLE"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="s3-secret-key">访问密钥</Label>
                <Input
                  id="s3-secret-key"
                  type="password"
                  value={s3Config.secret_access_key}
                  onChange={(e) => updateS3Config('secret_access_key', e.target.value)}
                  placeholder="wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
                />
              </div>
            </div>
          </CardContent>
        </Card>

        {/* 高级配置 */}
        <Card>
          <CardHeader>
            <CardTitle className="text-sm">高级配置</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="flex items-center space-x-2">
              <Checkbox
                id="s3-path-style"
                checked={s3Config.use_path_style}
                onCheckedChange={(checked) => updateS3Config('use_path_style', checked as boolean)}
              />
              <Label htmlFor="s3-path-style">使用Path Style URL</Label>
            </div>

            <div className="flex items-center space-x-2">
              <Checkbox
                id="s3-remove-bucket"
                checked={s3Config.remove_bucket}
                onCheckedChange={(checked) => updateS3Config('remove_bucket', checked as boolean)}
              />
              <Label htmlFor="s3-remove-bucket">从路径中删除bucket名称</Label>
            </div>

            <div className="space-y-2">
              <Label htmlFor="s3-custom-domain">自定义访问域名（可选）</Label>
              <Input
                id="s3-custom-domain"
                value={s3Config.custom_domain}
                onChange={(e) => updateS3Config('custom_domain', e.target.value)}
                placeholder="https://cdn.example.com"
              />
              <p className="text-xs text-muted-foreground">
                留空使用S3标准URL，支持路径如: https://cdn.example.com/path
              </p>
            </div>
          </CardContent>
        </Card>

        {/* 文件过滤配置 */}
        <Card>
          <CardHeader>
            <CardTitle className="text-sm">文件过滤配置</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="space-y-2">
              <Label htmlFor="s3-folder-path">文件夹路径（可选）</Label>
              <Input
                id="s3-folder-path"
                value={s3Config.folder_path}
                onChange={(e) => updateS3Config('folder_path', e.target.value)}
                placeholder="/images"
              />
              <p className="text-xs text-muted-foreground">
                指定要提取的文件夹路径，如: /images 或 /uploads/photos
              </p>
            </div>

            <div className="flex items-center space-x-2">
              <Checkbox
                id="s3-include-subfolders"
                checked={s3Config.include_subfolders}
                onCheckedChange={(checked) => updateS3Config('include_subfolders', checked as boolean)}
              />
              <Label htmlFor="s3-include-subfolders">包含所有子文件夹</Label>
            </div>

            <div className="space-y-2">
              <Label>文件格式过滤</Label>
              {extensionInputs.map((ext, index) => (
                <div key={index} className="flex gap-2">
                  <Input
                    value={ext}
                    onChange={(e) => updateExtension(index, e.target.value)}
                    placeholder=".jpg 或 .png"
                    className="flex-1"
                  />
                  {extensionInputs.length > 1 && (
                    <Button
                      type="button"
                      size="sm"
                      variant="outline"
                      onClick={() => removeExtension(index)}
                      className="text-red-600 hover:text-red-700"
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  )}
                </div>
              ))}
              <Button
                type="button"
                size="sm"
                variant="outline"
                onClick={addExtension}
                className="w-full"
              >
                <Plus className="h-4 w-4 mr-1" />
                添加文件格式
              </Button>
              <p className="text-xs text-muted-foreground">
                留空表示不过滤文件格式，支持正则匹配如: .jpg, .png, .gif
              </p>
            </div>
          </CardContent>
        </Card>
      </div>
    )
  }

  return null
} 