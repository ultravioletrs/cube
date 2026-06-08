// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

const AUDIT_INDEX = 'cube-audit-*'

function authHeaders(token: string, domainID?: string): Record<string, string> {
  const headers: Record<string, string> = {}
  if (token) headers.Authorization = `Bearer ${token}`
  if (domainID) headers['X-Domain-Id'] = domainID
  return headers
}

async function auditSearch(domainID: string, token: string, body: object): Promise<unknown> {
  const res = await fetch(`/proxy/${domainID}/audit/${AUDIT_INDEX}/_search`, {
    method: 'POST',
    credentials: 'omit',
    headers: { 'Content-Type': 'application/json', ...authHeaders(token, domainID) },
    body: JSON.stringify(body),
  })
  if (!res.ok) throw new Error(`OpenSearch error: ${res.status}`)
  return res.json()
}

export interface DashboardStats {
  conversationsToday: number
  activeModels: number
  tokensToday: number
}

export async function fetchDashboardStats(domainID: string, token: string): Promise<DashboardStats> {
  const osBody = {
    size: 0,
    query: {
      bool: {
        filter: [
          { range: { '@timestamp': { gte: 'now/d' } } },
          { terms: { event_type: ['llm_request', 'guardrails_request'] } },
          { range: { input_tokens: { gt: 0 } } },
        ],
      },
    },
    aggs: {
      active_models: { cardinality: { field: 'model.keyword' } },
      total_input_tokens: { sum: { field: 'input_tokens' } },
      total_output_tokens: { sum: { field: 'output_tokens' } },
    },
  }

  const [convRes, osData] = await Promise.all([
    fetch(`/${domainID}/api/v1/conversations`, {
      credentials: 'omit',
      headers: authHeaders(token, domainID),
    }),
    auditSearch(domainID, token, osBody),
  ])

  let conversationsToday = 0
  if (convRes.ok) {
    const convData = (await convRes.json()) as { conversations: Array<{ created_at: string }> }
    const todayStart = new Date()
    todayStart.setUTCHours(0, 0, 0, 0)
    conversationsToday = (convData.conversations ?? []).filter(
      c => new Date(c.created_at) >= todayStart,
    ).length
  }

  const aggs = (osData as any)?.aggregations ?? {}
  return {
    conversationsToday,
    activeModels: aggs.active_models?.value ?? 0,
    tokensToday: (aggs.total_input_tokens?.value ?? 0) + (aggs.total_output_tokens?.value ?? 0),
  }
}

export interface ActivityBucket {
  time: string
  sessions: number
  tokens: number
}

export async function fetchActivityTrends(domainID: string, token: string): Promise<ActivityBucket[]> {
  const now = new Date()
  const from = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000)
  const body = {
    size: 0,
    query: {
      bool: {
        filter: [
          { range: { '@timestamp': { gte: from.toISOString() } } },
          { terms: { event_type: ['llm_request', 'guardrails_request'] } },
        ],
      },
    },
    aggs: {
      activity_over_time: {
        date_histogram: {
          field: '@timestamp',
          fixed_interval: '3h',
          min_doc_count: 0,
          extended_bounds: { min: from.toISOString(), max: now.toISOString() },
        },
        aggs: {
          unique_sessions: { cardinality: { field: 'trace_id.keyword' } },
          input_tokens: { sum: { field: 'input_tokens' } },
          output_tokens: { sum: { field: 'output_tokens' } },
        },
      },
    },
  }
  const data = (await auditSearch(domainID, token, body)) as any
  const buckets = data?.aggregations?.activity_over_time?.buckets ?? []
  return buckets.map((b: any) => ({
    time: b.key_as_string ?? b.key,
    sessions: b.unique_sessions?.value ?? 0,
    tokens: (b.input_tokens?.value ?? 0) + (b.output_tokens?.value ?? 0),
  }))
}

export interface ErrorRateBucket {
  time: string
  success: number
  clientErrors: number
  serverErrors: number
}

export async function fetchErrorRateTrends(domainID: string, token: string): Promise<ErrorRateBucket[]> {
  const now = new Date()
  const from = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000)
  const body = {
    size: 0,
    query: {
      bool: {
        filter: [
          { range: { '@timestamp': { gte: from.toISOString() } } },
          { terms: { event_type: ['llm_request', 'guardrails_request'] } },
        ],
      },
    },
    aggs: {
      over_time: {
        date_histogram: {
          field: '@timestamp',
          fixed_interval: '3h',
          min_doc_count: 0,
          extended_bounds: { min: from.toISOString(), max: now.toISOString() },
        },
        aggs: {
          success: { filter: { range: { status_code: { gte: 200, lt: 300 } } } },
          client_errors: { filter: { range: { status_code: { gte: 400, lt: 500 } } } },
          server_errors: { filter: { range: { status_code: { gte: 500 } } } },
        },
      },
    },
  }
  const data = (await auditSearch(domainID, token, body)) as any
  const buckets = data?.aggregations?.over_time?.buckets ?? []
  return buckets.map((b: any) => ({
    time: b.key_as_string ?? b.key,
    success: b.success?.doc_count ?? 0,
    clientErrors: b.client_errors?.doc_count ?? 0,
    serverErrors: b.server_errors?.doc_count ?? 0,
  }))
}

export interface ModelPerf {
  model: string
  requestCount: number
  avgLatencyMs: number
  avgInputTokens: number
  avgOutputTokens: number
}

export async function fetchModelPerformance(domainID: string, token: string): Promise<ModelPerf[]> {
  const body = {
    size: 0,
    query: {
      bool: {
        filter: [
          { range: { '@timestamp': { gte: 'now-7d' } } },
          { terms: { event_type: ['llm_request', 'guardrails_request'] } },
        ],
      },
    },
    aggs: {
      by_model: {
        terms: { field: 'model.keyword', size: 10 },
        aggs: {
          avg_latency: { avg: { field: 'duration_ms' } },
          avg_input_tokens: { avg: { field: 'input_tokens' } },
          avg_output_tokens: { avg: { field: 'output_tokens' } },
          request_count: { value_count: { field: '_id' } },
        },
      },
    },
  }
  const data = (await auditSearch(domainID, token, body)) as any
  const buckets = data?.aggregations?.by_model?.buckets ?? []
  return buckets.map((b: any) => ({
    model: b.key,
    requestCount: b.request_count?.value ?? 0,
    avgLatencyMs: Math.round(b.avg_latency?.value ?? 0),
    avgInputTokens: Math.round(b.avg_input_tokens?.value ?? 0),
    avgOutputTokens: Math.round(b.avg_output_tokens?.value ?? 0),
  }))
}

export interface TokenBreakdown {
  model: string
  inputTokens: number
  outputTokens: number
  totalTokens: number
  percentage: number
}

export async function fetchTokenBreakdown(domainID: string, token: string): Promise<TokenBreakdown[]> {
  const body = {
    size: 0,
    query: {
      bool: {
        filter: [
          { range: { '@timestamp': { gte: 'now-7d' } } },
          { terms: { event_type: ['llm_request', 'guardrails_request'] } },
        ],
      },
    },
    aggs: {
      by_model: {
        terms: { field: 'model.keyword', size: 10 },
        aggs: {
          input_tokens: { sum: { field: 'input_tokens' } },
          output_tokens: { sum: { field: 'output_tokens' } },
        },
      },
    },
  }
  const data = (await auditSearch(domainID, token, body)) as any
  const buckets = data?.aggregations?.by_model?.buckets ?? []
  let totalAll = 0
  const items = buckets.map((b: any) => {
    const inp = b.input_tokens?.value ?? 0
    const out = b.output_tokens?.value ?? 0
    const total = inp + out
    totalAll += total
    return { model: b.key, inputTokens: inp, outputTokens: out, totalTokens: total, percentage: 0 }
  })
  return items.map((item: TokenBreakdown) => ({
    ...item,
    percentage: totalAll > 0 ? (item.totalTokens / totalAll) * 100 : 0,
  }))
}

export interface GuardrailsStats {
  totalRequests: number
  cleanRequests: number
  contentFiltered: number
  piiDetected: number
  guardrailsProcessed: number
  decisionsAllow: number
  decisionsBlock: number
  decisionsModify: number
  promptInjection: number
  jailbreakAttempt: number
  toxicContent: number
  offTopicDetected: number
  hallucinationRisk: number
  avgGuardrailsLatencyMs: number
}

export async function fetchGuardrailsActivity(domainID: string, token: string): Promise<GuardrailsStats> {
  const body = {
    size: 0,
    query: {
      bool: {
        filter: [
          { range: { '@timestamp': { gte: 'now-7d' } } },
          { terms: { event_type: ['llm_request', 'guardrails_request'] } },
        ],
      },
    },
    aggs: {
      total_requests: { value_count: { field: '_id' } },
      content_filtered: { filter: { term: { content_filtered: true } } },
      pii_detected: { filter: { term: { pii_detected: true } } },
      guardrails_processed: { filter: { term: { guardrails_processed: true } } },
      decisions_allow: { filter: { term: { 'guardrails_decision.keyword': 'ALLOW' } } },
      decisions_block: { filter: { term: { 'guardrails_decision.keyword': 'BLOCK' } } },
      decisions_modify: { filter: { term: { 'guardrails_decision.keyword': 'MODIFY' } } },
      prompt_injection: { filter: { term: { prompt_injection: true } } },
      jailbreak_attempt: { filter: { term: { jailbreak_attempt: true } } },
      toxic_content: { filter: { term: { toxic_content: true } } },
      off_topic_detected: { filter: { term: { off_topic_detected: true } } },
      hallucination_risk: { filter: { term: { hallucination_risk: true } } },
      avg_guardrails_latency: { avg: { field: 'guardrails_latency_ms' } },
    },
  }
  const data = (await auditSearch(domainID, token, body)) as any
  const a = data?.aggregations ?? {}
  return {
    totalRequests: a.total_requests?.value ?? 0,
    cleanRequests:
      (a.total_requests?.value ?? 0) -
      (a.content_filtered?.doc_count ?? 0) -
      (a.pii_detected?.doc_count ?? 0),
    contentFiltered: a.content_filtered?.doc_count ?? 0,
    piiDetected: a.pii_detected?.doc_count ?? 0,
    guardrailsProcessed: a.guardrails_processed?.doc_count ?? 0,
    decisionsAllow: a.decisions_allow?.doc_count ?? 0,
    decisionsBlock: a.decisions_block?.doc_count ?? 0,
    decisionsModify: a.decisions_modify?.doc_count ?? 0,
    promptInjection: a.prompt_injection?.doc_count ?? 0,
    jailbreakAttempt: a.jailbreak_attempt?.doc_count ?? 0,
    toxicContent: a.toxic_content?.doc_count ?? 0,
    offTopicDetected: a.off_topic_detected?.doc_count ?? 0,
    hallucinationRisk: a.hallucination_risk?.doc_count ?? 0,
    avgGuardrailsLatencyMs: a.avg_guardrails_latency?.value ?? 0,
  }
}

export function formatTokens(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}k`
  return n.toString()
}
