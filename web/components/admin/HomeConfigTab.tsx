'use client'

import { useState, useEffect } from 'react'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'

interface HomeConfigTabProps {
  config: string
  onUpdate: (content: string) => void
}

export default function HomeConfigTab({ config, onUpdate }: HomeConfigTabProps) {
  const [content, setContent] = useState(config)

  useEffect(() => {
    setContent(config)
  }, [config])

  const handleSave = () => {
    onUpdate(content)
  }

  const handleReset = () => {
    setContent(config)
  }

  const hasChanges = content !== config

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <h2 className="text-2xl font-bold tracking-tight">首页配置</h2>
        <div className="flex space-x-2">
          {hasChanges && (
            <Button
              onClick={handleReset}
              variant="outline"
            >
              重置
            </Button>
          )}
          <Button
            onClick={handleSave}
            disabled={!hasChanges}
          >
            保存配置
          </Button>
        </div>
      </div>

      <div className="bg-card rounded-lg border p-6">
        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="markdown-content">Markdown内容</Label>
            <Textarea
              id="markdown-content"
              value={content}
              onChange={(e) => setContent(e.target.value)}
              className="min-h-96 font-mono text-sm"
              placeholder="输入Markdown格式的内容..."
            />
          </div>
          
          <div className="bg-muted rounded-lg p-4">
            <h4 className="text-sm font-medium mb-2">使用说明</h4>
            <ul className="text-sm text-muted-foreground space-y-1">
              <li>• 支持标准Markdown语法，包括标题、列表、粗体、斜体等</li>
              <li>• 可以使用代码块来展示API使用示例</li>
              <li>• 支持链接和图片嵌入</li>
              <li>• 内容将显示在首页的介绍区域</li>
            </ul>
          </div>

          {hasChanges && (
            <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-4">
              <div className="flex items-center">
                <div className="flex-shrink-0">
                  <svg className="h-5 w-5 text-yellow-400" viewBox="0 0 20 20" fill="currentColor">
                    <path fillRule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clipRule="evenodd" />
                  </svg>
                </div>
                <div className="ml-3">
                  <p className="text-sm text-yellow-800">
                    您有未保存的更改，请点击&quot;保存配置&quot;按钮保存修改。
                  </p>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
} 