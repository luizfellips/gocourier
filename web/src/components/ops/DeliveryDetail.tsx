import { AlertCircle, Check, Copy, RotateCw, X } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { useDeliveryDetail, useReplay } from "@/hooks/useDashboard";
import { useAppStore } from "@/stores/appStore";
import { cn } from "@/lib/utils";
import { StatusBadge } from "./StatusBadge";

function eventDotTone(type: string) {
  if (type.includes("Succeeded")) return "bg-status-success";
  if (type.includes("DLQ") || type.includes("Failed") || type.includes("Rejected")) return "bg-status-danger";
  if (type.includes("Retrying") || type.includes("Attempted") || type.includes("Queued") || type.includes("Replayed")) return "bg-status-warning";
  return "bg-status-neutral";
}

function fmt(iso: string) {
  return new Date(iso).toISOString().replace("T", " ").replace("Z", "");
}

export function DeliveryDetail() {
  const id = useAppStore((s) => s.selectedDeliveryId);
  const close = () => useAppStore.getState().setSelectedDeliveryId(null);
  const { data, isLoading } = useDeliveryDetail(id);
  const replay = useReplay();

  if (!id) {
    return (
      <aside className="hidden h-full flex-col items-center justify-center gap-2 rounded-lg border border-dashed border-border bg-card/40 p-6 text-center text-xs text-muted-foreground lg:flex">
        <div className="mono">No delivery selected</div>
        <div>Click a row in the table or the activity feed</div>
      </aside>
    );
  }

  if (isLoading || !data) {
    return (
      <aside className="rounded-lg border border-border bg-card p-4 text-xs text-muted-foreground">
        Loading delivery…
      </aside>
    );
  }

  const { delivery, attempts, audit, outbox } = data;
  const sortedAudit = [...audit].sort(
    (a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime(),
  );

  function copy() {
    navigator.clipboard.writeText(delivery.id).then(() => toast.success("Copied delivery id"));
  }

  function doReplay() {
    replay.mutate(delivery.id, {
      onSuccess: () => toast.success("Replay queued"),
      onError: (e) => toast.error("Replay failed", { description: (e as Error).message }),
    });
  }

  return (
    <aside className="flex h-full min-h-0 flex-col overflow-hidden rounded-lg border border-border bg-card">
      <div className="flex items-start justify-between gap-2 border-b border-border px-3 py-2.5">
        <div className="min-w-0 space-y-1.5">
          <div className="flex items-center gap-2">
            <StatusBadge status={delivery.status} />
            <span className="mono text-[11px] text-muted-foreground uppercase">
              {delivery.channel} · {delivery.priority}
            </span>
          </div>
          <div className="flex items-center gap-1">
            <code className="block truncate mono text-[11px] text-foreground/80">{delivery.id}</code>
            <button onClick={copy} className="rounded p-1 text-muted-foreground hover:text-foreground" title="Copy id">
              <Copy className="h-3 w-3" />
            </button>
          </div>
        </div>
        <div className="flex items-center gap-1">
          {delivery.status === "dlq" && (
            <Button size="sm" variant="default" className="h-7 text-[11px]" onClick={doReplay} disabled={replay.isPending}>
              <RotateCw className={cn("mr-1 h-3 w-3", replay.isPending && "animate-spin")} />
              Replay
            </Button>
          )}
          <button onClick={close} className="rounded p-1 text-muted-foreground hover:text-foreground" title="Close">
            <X className="h-4 w-4" />
          </button>
        </div>
      </div>

      <ScrollArea className="min-h-0 flex-1">
        <div className="space-y-4 p-3">
          {delivery.last_error && (
            <div className="rounded-md border border-status-danger/30 bg-status-danger/10 p-2.5">
              <div className="mb-1 flex items-center gap-1.5 text-[11px] font-medium uppercase tracking-wider text-status-danger">
                <AlertCircle className="h-3 w-3" /> Last error
              </div>
              <code className="block mono text-[11px] text-status-danger break-words">
                {delivery.last_error}
              </code>
            </div>
          )}

          <section>
            <SectionLabel>Timeline</SectionLabel>
            <ol className="relative ml-2 mt-2 space-y-3 border-l border-border pl-4">
              {sortedAudit.map((a) => (
                <li key={a.id} className="relative">
                  <span
                    className={cn(
                      "absolute -left-[21px] top-1 grid h-3 w-3 place-items-center rounded-full ring-2 ring-card",
                      eventDotTone(a.event_type),
                    )}
                  />
                  <div className="mono text-xs text-foreground">{a.event_type}</div>
                  <div className="mono text-[10px] text-muted-foreground">{fmt(a.created_at)}</div>
                </li>
              ))}
              {sortedAudit.length === 0 && <li className="text-[11px] text-muted-foreground">No audit events</li>}
            </ol>
          </section>

          <section>
            <SectionLabel>Dispatch attempts ({attempts.length})</SectionLabel>
            <ul className="mt-2 space-y-1.5">
              {attempts.map((a) => (
                <li
                  key={a.id}
                  className="flex items-start gap-2 rounded-md border border-border bg-background/40 p-2"
                >
                  <span
                    className={cn(
                      "mt-0.5 grid h-4 w-4 shrink-0 place-items-center rounded-full",
                      a.success ? "bg-status-success/15 text-status-success" : "bg-status-danger/15 text-status-danger",
                    )}
                  >
                    {a.success ? <Check className="h-2.5 w-2.5" /> : <X className="h-2.5 w-2.5" />}
                  </span>
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center justify-between gap-2">
                      <span className="mono text-[11px]">attempt #{a.attempt_number}</span>
                      <span className="mono text-[10px] text-muted-foreground">{fmt(a.started_at)}</span>
                    </div>
                    {!a.success && a.error_message && (
                      <div className="mt-0.5 mono text-[11px] text-status-danger break-words">{a.error_message}</div>
                    )}
                  </div>
                </li>
              ))}
              {attempts.length === 0 && <li className="text-[11px] text-muted-foreground">No attempts yet</li>}
            </ul>
          </section>

          <section>
            <SectionLabel>Outbox ({outbox.length})</SectionLabel>
            <ul className="mt-2 space-y-1.5">
              {outbox.map((o) => (
                <li
                  key={o.id}
                  className="flex items-start justify-between gap-2 rounded-md border border-border bg-background/40 p-2"
                >
                  <div className="min-w-0">
                    <div className="mono text-[11px] text-foreground truncate">{o.subject}</div>
                    <div className="mono text-[10px] text-muted-foreground">
                      created {fmt(o.created_at)}
                      {o.published_at && ` · published ${fmt(o.published_at)}`}
                    </div>
                  </div>
                  <span
                    className={cn(
                      "mono text-[10px] uppercase tracking-wider",
                      o.status === "published" && "text-status-success",
                      o.status === "pending" && "text-status-warning",
                      o.status === "failed" && "text-status-danger",
                    )}
                  >
                    {o.status}
                  </span>
                </li>
              ))}
              {outbox.length === 0 && <li className="text-[11px] text-muted-foreground">No outbox rows</li>}
            </ul>
          </section>
        </div>
      </ScrollArea>
    </aside>
  );
}

function SectionLabel({ children }: { children: React.ReactNode }) {
  return (
    <div className="text-[10px] font-medium uppercase tracking-wider text-muted-foreground">{children}</div>
  );
}