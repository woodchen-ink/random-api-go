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
  type: 'lankong' | 'manual' | 'api_get' | 'api_post' | 'endpoint' | 's3'
  config: string
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

export interface DomainStatsResult {
  domain: string
  count: number
  is_blocked: boolean
}

export interface DomainStatsData {
  top_24_hours: DomainStatsResult[]
  top_7_days: DomainStatsResult[]
  top_30_days: DomainStatsResult[]
  top_total: DomainStatsResult[]
}

export interface DomainPathStat {
  path: string
  count: number
}

export interface DomainPathStatsResponse {
  domain: string
  range: string
  paths: DomainPathStat[]
}

export interface DomainDailyPoint {
  date: string
  count: number
}

export interface DomainTrendData {
  days: number
  series: DomainDailyPoint[]
}

export interface BlockedDomain {
  id: number
  domain: string
  reason: string
  created_at: string
  updated_at: string
}
