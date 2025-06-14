export interface User {
  id: string
  name: string
  email: string
}

export interface APIEndpoint {
  id: number
  name: string
  url: string
  description: string
  is_active: boolean
  show_on_homepage: boolean
  sort_order: number
  created_at: string
  updated_at: string
  data_sources?: DataSource[]
}

export interface DataSource {
  id: number
  endpoint_id: number
  name: string
  type: 'lankong' | 'manual' | 'api_get' | 'api_post' | 'endpoint'
  config: string
  cache_duration: number
  is_active: boolean
  last_sync?: string
  created_at: string
  updated_at: string
}

export interface URLReplaceRule {
  id: number
  endpoint_id?: number
  name: string
  from_url: string
  to_url: string
  is_active: boolean
  created_at: string
  updated_at: string
  endpoint?: APIEndpoint
}

export interface OAuthConfig {
  client_id: string
  base_url: string
} 