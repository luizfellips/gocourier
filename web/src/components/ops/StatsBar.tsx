import { AlertTriangle, Clock, Database, Layers } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import type { DashboardSummary, DeliveryStatus } from "@/api/types";
import { StatusBadge } from "./StatusBadge";

const STATUS_ORDER: DeliveryStatus[] = [
  "succeeded",
  "processing",
  "queued",
  "retrying",
  "pending",
  "dlq",
  "failed",
  "rejected",
  "cancelled",
];

function useStaleSeconds(value: number) {
  const [seconds, setSeconds] = useState(0);
  const startedAt = useRef<number | null>(null);
  useEffect(() => {
    if (value > 0) {
      if (startedAt.current === null) startedAt.current = Date.now();
      const i = setInterval(() => {
        setSeconds(Math.floor((Date.now() - (startedAt.current ?? Date.now())) / 1000));
      }, 1000);
      return () => clearInterval(i);
    }
    startedAt.current = null;
    setSeconds(0);
  }, [value]);
  return seconds;
}

export function StatsBar({ summary }: { summary: DashboardSummary | undefined }) {
  const outboxPending = summary?.outbox_pending ?? 0;
  const scheduledDue = summary?.scheduled_due ?? 0;
  const total = summary?.deliveries?.length ?? 0;
  const stale = useStaleSeconds(outboxPending);
  const outboxAlert = outboxPending > 0 && stale > 30;

  return (
    <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
      <Kpi
        label="Outbox pending"
        value={outboxPending}
        icon={<Database className="h-3.5 w-3.5" />}
        tone={outboxAlert ? "danger" : outboxPending > 0 ? "warning" : "neutral"}
        hint={outboxAlert ? `stuck ${stale}s` : "publish backlog"}
      />
      <Kpi
        label="Scheduled due"
        value={scheduledDue}
        icon={<Clock className="h-3.5 w-3.5" />}
        tone={scheduledDue > 0 ? "warning" : "neutral"}
        hint="ready to fire"
      />
      <Kpi
        label="Total deliveries"
        value={total}
        icon={<Layers className="h-3.5 w-3.5" />}
        tone="neutral"
        hint="in current window"
      />
      <div className="rounded-lg border border-border bg-card p-3">
        <div className="mb-2 flex items-center gap-2 text-[11px] uppercase tracking-wider text-muted-foreground">
          <AlertTriangle className="h-3.5 w-3.5" /> By status
        </div>
        <div className="flex flex-wrap gap-1.5">
          {STATUS_ORDER.map((s) => {
            const c = summary?.status_counts?.[s] ?? 0;
            if (!c) return null;
            return (
              <span key={s} className="inline-flex items-center gap-1.5">
                <StatusBadge status={s} />
                <span className="mono text-xs text-foreground">{c}</span>
              </span>
            );
          })}
          {STATUS_ORDER.every((s) => !(summary?.status_counts?.[s])) && (
            <span className="text-xs text-muted-foreground">No deliveries</span>
          )}
        </div>
      </div>
    </div>
  );
}

function Kpi({
  label,
  value,
  icon,
  tone,
  hint,
}: {
  label: string;
  value: number;
  icon: React.ReactNode;
  tone: "success" | "warning" | "danger" | "neutral";
  hint?: string;
}) {
  const color =
    tone === "danger"
      ? "text-status-danger"
      : tone === "warning"
      ? "text-status-warning"
      : tone === "success"
      ? "text-status-success"
      : "text-foreground";
  return (
    <div className="rounded-lg border border-border bg-card p-3">
      <div className="mb-1 flex items-center gap-2 text-[11px] uppercase tracking-wider text-muted-foreground">
        {icon}
        {label}
      </div>
      <div className={`mono text-2xl font-semibold tabular-nums ${color}`}>{value}</div>
      {hint && <div className="text-[11px] text-muted-foreground mt-0.5">{hint}</div>}
    </div>
  );
}