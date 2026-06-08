import { ArrowDownUp, Inbox } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import type { Channel, DeliveryRow, DeliveryStatus } from "@/api/types";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useAppStore } from "@/stores/appStore";
import { cn } from "@/lib/utils";
import { StatusBadge } from "./StatusBadge";

function relative(iso: string, now: number) {
  const s = Math.max(0, Math.floor((now - new Date(iso).getTime()) / 1000));
  if (s < 2) return "now";
  if (s < 60) return `${s}s ago`;
  const m = Math.floor(s / 60);
  if (m < 60) return `${m}m ago`;
  return `${Math.floor(m / 60)}h ago`;
}

export function DeliveriesTable({ rows }: { rows: DeliveryRow[] }) {
  const selected = useAppStore((s) => s.selectedDeliveryId);
  const setSelected = useAppStore((s) => s.setSelectedDeliveryId);
  const [statusFilter, setStatusFilter] = useState<DeliveryStatus | "all">("all");
  const [channelFilter, setChannelFilter] = useState<Channel | "all">("all");
  const [sortDesc, setSortDesc] = useState(true);
  const [now, setNow] = useState(Date.now());
  useEffect(() => {
    const i = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(i);
  }, []);

  const filtered = useMemo(() => {
    let r = rows.slice();
    if (statusFilter !== "all") r = r.filter((x) => x.status === statusFilter);
    if (channelFilter !== "all") r = r.filter((x) => x.channel === channelFilter);
    r.sort((a, b) => {
      const d = new Date(b.created_at).getTime() - new Date(a.created_at).getTime();
      return sortDesc ? d : -d;
    });
    return r;
  }, [rows, statusFilter, channelFilter, sortDesc]);

  return (
    <div className="flex min-h-0 flex-1 flex-col rounded-lg border border-border bg-card">
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-border px-3 py-2">
        <div className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
          Deliveries <span className="ml-2 mono text-[10px] normal-case text-foreground/60">{filtered.length} / {rows.length}</span>
        </div>
        <div className="flex items-center gap-2">
          <Select value={statusFilter} onValueChange={(v) => setStatusFilter(v as DeliveryStatus | "all")}>
            <SelectTrigger className="h-7 w-32 text-[11px] mono"><SelectValue placeholder="status" /></SelectTrigger>
            <SelectContent>
              <SelectItem value="all">all statuses</SelectItem>
              {["pending","queued","processing","retrying","succeeded","failed","dlq","cancelled","rejected"].map((s) => (
                <SelectItem key={s} value={s}>{s}</SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Select value={channelFilter} onValueChange={(v) => setChannelFilter(v as Channel | "all")}>
            <SelectTrigger className="h-7 w-28 text-[11px] mono"><SelectValue /></SelectTrigger>
            <SelectContent>
              <SelectItem value="all">all channels</SelectItem>
              <SelectItem value="email">email</SelectItem>
              <SelectItem value="sms">sms</SelectItem>
              <SelectItem value="push">push</SelectItem>
              <SelectItem value="webhook">webhook</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </div>

      <div className="min-h-0 flex-1 overflow-auto">
        {filtered.length === 0 ? (
          <div className="flex h-full flex-col items-center justify-center gap-2 py-12 text-muted-foreground">
            <Inbox className="h-6 w-6" />
            <div className="text-xs">No deliveries yet — send a test notification</div>
          </div>
        ) : (
          <table className="w-full border-collapse text-xs">
            <thead className="sticky top-0 z-10 bg-card">
              <tr className="text-left text-[10px] uppercase tracking-wider text-muted-foreground">
                <th className="border-b border-border px-3 py-2 font-medium">
                  <button onClick={() => setSortDesc((v) => !v)} className="inline-flex items-center gap-1 hover:text-foreground">
                    Time <ArrowDownUp className="h-3 w-3" />
                  </button>
                </th>
                <th className="border-b border-border px-3 py-2 font-medium">Channel</th>
                <th className="border-b border-border px-3 py-2 font-medium">Status</th>
                <th className="border-b border-border px-3 py-2 font-medium">Idempotency Key</th>
                <th className="border-b border-border px-3 py-2 text-right font-medium">Retries</th>
              </tr>
            </thead>
            <tbody>
              {filtered.map((d) => (
                <tr
                  key={d.id}
                  onClick={() => setSelected(d.id)}
                  className={cn(
                    "cursor-pointer border-b border-border/50 transition-colors hover:bg-accent/40",
                    selected === d.id && "bg-accent/60",
                  )}
                >
                  <td className="px-3 py-2 mono text-muted-foreground whitespace-nowrap">{relative(d.created_at, now)}</td>
                  <td className="px-3 py-2 mono">{d.channel}</td>
                  <td className="px-3 py-2"><StatusBadge status={d.status} /></td>
                  <td className="px-3 py-2 mono text-foreground/80">
                    <span title={d.idempotency_key}>
                      {d.idempotency_key.length > 32 ? d.idempotency_key.slice(0, 30) + "…" : d.idempotency_key}
                    </span>
                  </td>
                  <td className={cn("px-3 py-2 text-right mono tabular-nums", d.retry_count > 0 && "text-status-warning")}>
                    {d.retry_count}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}