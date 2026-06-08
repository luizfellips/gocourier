import { useEffect, useState } from "react";
import type { ActivityEvent } from "@/api/types";
import { useAppStore } from "@/stores/appStore";
import { cn } from "@/lib/utils";

function relative(iso: string, now: number) {
  const s = Math.max(0, Math.floor((now - new Date(iso).getTime()) / 1000));
  if (s < 2) return "now";
  if (s < 60) return `${s}s`;
  const m = Math.floor(s / 60);
  if (m < 60) return `${m}m`;
  return `${Math.floor(m / 60)}h`;
}

function eventTone(type: string): string {
  if (type.includes("Succeeded")) return "text-status-success";
  if (type.includes("DLQ") || type.includes("Failed") || type.includes("Rejected")) return "text-status-danger";
  if (type.includes("Retrying") || type.includes("Attempted") || type.includes("Queued")) return "text-status-warning";
  return "text-muted-foreground";
}

export function ActivityFeed({ events }: { events: ActivityEvent[] }) {
  const setSelected = useAppStore((s) => s.setSelectedDeliveryId);
  const selected = useAppStore((s) => s.selectedDeliveryId);
  const [now, setNow] = useState(Date.now());
  useEffect(() => {
    const i = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(i);
  }, []);

  return (
    <div className="flex min-h-0 flex-col rounded-lg border border-border bg-card">
      <div className="flex items-center justify-between border-b border-border px-3 py-2">
        <div className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
          Activity
        </div>
        <div className="text-[10px] text-muted-foreground mono">{events.length} events</div>
      </div>
      <div className="max-h-[420px] overflow-y-auto">
        {events.length === 0 ? (
          <div className="px-3 py-6 text-center text-xs text-muted-foreground">No activity yet</div>
        ) : (
          <ul className="divide-y divide-border">
            {events.map((e) => (
              <li
                key={e.id}
                onClick={() => setSelected(e.delivery_id)}
                className={cn(
                  "cursor-pointer px-3 py-2 transition-colors hover:bg-accent/50",
                  selected === e.delivery_id && "bg-accent/40",
                )}
              >
                <div className="flex items-center justify-between gap-2">
                  <span className={cn("text-xs mono truncate", eventTone(e.event_type))}>
                    {e.event_type}
                  </span>
                  <span className="shrink-0 text-[10px] text-muted-foreground mono">
                    {relative(e.created_at, now)}
                  </span>
                </div>
                <div className="mt-0.5 truncate text-[11px] text-muted-foreground mono">
                  {e.delivery_id.slice(0, 8)}…
                </div>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}