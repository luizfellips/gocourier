export type Channel = "email" | "sms" | "push" | "webhook";
export type Priority = "low" | "normal" | "high";
export type DeliveryStatus =
  | "pending"
  | "queued"
  | "processing"
  | "retrying"
  | "succeeded"
  | "failed"
  | "dlq"
  | "cancelled"
  | "rejected";

export interface DeliveryRow {
  id: string;
  idempotency_key: string;
  channel: Channel;
  priority: Priority;
  status: DeliveryStatus;
  retry_count: number;
  last_error: string;
  created_at: string;
  updated_at: string;
}

export interface ActivityEvent {
  id: number;
  delivery_id: string;
  event_type: string;
  payload: Record<string, unknown>;
  created_at: string;
}

export interface DashboardSummary {
  status_counts: Partial<Record<DeliveryStatus, number>>;
  outbox_pending: number;
  scheduled_due: number;
  recent_activity: ActivityEvent[];
  deliveries: DeliveryRow[];
}

export interface AttemptRow {
  id: string;
  attempt_number: number;
  success: boolean;
  error_message: string;
  started_at: string;
  finished_at: string;
  provider_response: Record<string, unknown>;
}

export interface AuditRow {
  id: number;
  delivery_id: string;
  event_type: string;
  created_at: string;
}

export interface OutboxRow {
  id: number;
  status: "pending" | "published" | "failed";
  subject: string;
  attempts: number;
  created_at: string;
  published_at: string | null;
}

export interface DeliveryDetailResp {
  delivery: DeliveryRow;
  attempts: AttemptRow[];
  audit: AuditRow[];
  outbox: OutboxRow[];
}

export interface HealthResp {
  status: "ok" | "down";
  time: string;
}

export interface SendNotificationReq {
  schema_version: string;
  idempotency_key: string;
  channel: Channel;
  priority: Priority;
  recipient: { address: string };
  template: { id: string; data: Record<string, unknown> };
}

export interface SendNotificationResp {
  delivery_id: string;
  status: DeliveryStatus;
  duplicate?: boolean;
}