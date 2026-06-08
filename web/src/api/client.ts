import type {
  ActivityEvent,
  AttemptRow,
  AuditRow,
  DashboardSummary,
  DeliveryDetailResp,
  DeliveryRow,
  DeliveryStatus,
  HealthResp,
  OutboxRow,
  SendNotificationReq,
  SendNotificationResp,
} from './types'

const API_BASE = import.meta.env.VITE_API_BASE ?? ''

async function request<T>(
  path: string,
  init: RequestInit & { apiKey?: string } = {},
): Promise<T> {
  const { apiKey, headers, ...rest } = init
  const res = await fetch(`${API_BASE}${path}`, {
    ...rest,
    headers: {
      'Content-Type': 'application/json',
      ...(apiKey ? { 'X-API-Key': apiKey } : {}),
      ...headers,
    },
  })
  if (!res.ok) {
    const body = await res.json().catch(() => ({}))
    throw new Error((body as { error?: string }).error ?? res.statusText)
  }
  return res.json() as Promise<T>
}

function asDeliveryRow(d: DeliveryRow): DeliveryRow {
  return {
    ...d,
    last_error: d.last_error ?? '',
    priority: d.priority ?? 'normal',
  }
}

function asAttempt(a: AttemptRow): AttemptRow {
  return {
    ...a,
    error_message: a.error_message ?? '',
    finished_at: a.finished_at ?? a.started_at,
    provider_response: a.provider_response ?? {},
  }
}

function asAudit(e: AuditRow & { payload?: unknown }): AuditRow {
  return {
    id: e.id,
    delivery_id: e.delivery_id,
    event_type: e.event_type,
    created_at: e.created_at,
  }
}

function asActivity(e: ActivityEvent & { payload?: unknown }): ActivityEvent {
  const payload =
    e.payload && typeof e.payload === 'object' && !Array.isArray(e.payload)
      ? (e.payload as Record<string, unknown>)
      : {}
  return {
    id: e.id,
    delivery_id: e.delivery_id,
    event_type: e.event_type,
    payload,
    created_at: e.created_at,
  }
}

function normalizeSummary(raw: DashboardSummary): DashboardSummary {
  return {
    status_counts: raw.status_counts ?? {},
    outbox_pending: raw.outbox_pending ?? 0,
    scheduled_due: raw.scheduled_due ?? 0,
    recent_activity: (raw.recent_activity ?? []).map(asActivity),
    deliveries: (raw.deliveries ?? []).map(asDeliveryRow),
  }
}

function normalizeDetail(raw: DeliveryDetailResp): DeliveryDetailResp {
  return {
    delivery: asDeliveryRow(raw.delivery),
    attempts: (raw.attempts ?? []).map(asAttempt),
    audit: (raw.audit ?? []).map(asAudit),
    outbox: (raw.outbox ?? []).map((o) => ({
      ...o,
      published_at: o.published_at ?? null,
      status: o.status as OutboxRow['status'],
    })),
  }
}

export async function getHealth(): Promise<HealthResp> {
  const data = await request<{ status: string; time: string }>('/health')
  return { status: data.status === 'ok' ? 'ok' : 'down', time: data.time }
}

export async function getSummary(limit = 30): Promise<DashboardSummary> {
  const data = await request<DashboardSummary>(`/v1/dashboard/summary?limit=${limit}`)
  return normalizeSummary(data)
}

export async function getDelivery(id: string): Promise<DeliveryDetailResp> {
  const data = await request<DeliveryDetailResp>(`/v1/dashboard/deliveries/${id}`)
  return normalizeDetail(data)
}

export async function sendNotification(
  req: SendNotificationReq,
  apiKey: string,
): Promise<SendNotificationResp> {
  if (!apiKey) throw new Error('Missing X-API-Key')
  const data = await request<SendNotificationResp>('/v1/notifications', {
    method: 'POST',
    apiKey,
    body: JSON.stringify(req),
  })
  return {
    ...data,
    status: data.status as DeliveryStatus,
  }
}

export async function replayDelivery(
  id: string,
  apiKey: string,
): Promise<SendNotificationResp> {
  if (!apiKey) throw new Error('Missing X-API-Key')
  const data = await request<{ delivery_id: string; status: string }>(
    `/v1/notifications/${id}/replay`,
    { method: 'POST', apiKey },
  )
  return {
    delivery_id: data.delivery_id,
    status: data.status as DeliveryStatus,
  }
}
