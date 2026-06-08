import { AlertCircle, Check, Copy, RotateCw, Search, X } from "lucide-react";
import { useEffect, useState } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
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

function fmtJSON(value: Record<string, unknown> | null | undefined) {
  if (!value || Object.keys(value).length === 0) return null;
  return JSON.stringify(value, null, 2);
}

const UUID_RE =
  /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;

export function DeliveryDetail() {
  const id = useAppStore((s) => s.selectedDeliveryId);
  const setSelected = useAppStore((s) => s.setSelectedDeliveryId);
  const close = () => setSelected(null);
  const { data, isLoading, isError, error } = useDeliveryDetail(id);
  const replay = useReplay();
  const [lookup, setLookup] = useState("");

  useEffect(() => {
    setLookup(id ?? "");
  }, [id]);

  function submitLookup(e?: React.FormEvent) {
    e?.preventDefault();
    const trimmed = lookup.trim();
    if (!trimmed) {
      toast.error("Enter a delivery id");
      return;
    }
    if (!UUID_RE.test(trimmed)) {
      toast.error("Invalid delivery id", { description: "Expected a UUID like 1d0973f2-5512-494c-b767-c44614b08…" });
      return;
    }
    setSelected(trimmed);
  }

  const lookupBar = (
    <form onSubmit={submitLookup} className="border-b border-border px-3 py-2.5">
      <div className="mb-1.5 text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
        Lookup delivery
      </div>
      <div className="flex gap-1.5">
        <Input
          value={lookup}
          onChange={(e) => setLookup(e.target.value)}
          placeholder="Paste delivery UUID…"
          className="h-8 mono text-[11px]"
          spellCheck={false}
        />
        <Button type="submit" size="sm" className="h-8 shrink-0 px-2.5" disabled={isLoading && !!id}>
          <Search className="h-3.5 w-3.5" />
        </Button>
      </div>
    </form>
  );

  if (!id) {
    return (
      <aside className="flex h-full min-h-0 flex-col overflow-hidden rounded-lg border border-border bg-card">
        {lookupBar}
        <div className="flex flex-1 flex-col items-center justify-center gap-2 p-6 text-center text-xs text-muted-foreground">
          <div className="mono">No delivery selected</div>
          <div>Paste an id above, or click a row in the table / activity feed</div>
        </div>
      </aside>
    );
  }

  if (isLoading && !data) {
    return (
      <aside className="flex h-full min-h-0 flex-col overflow-hidden rounded-lg border border-border bg-card">
        {lookupBar}
        <div className="p-4 text-xs text-muted-foreground">Loading delivery…</div>
      </aside>
    );
  }

  if (isError || !data) {
    return (
      <aside className="flex h-full min-h-0 flex-col overflow-hidden rounded-lg border border-border bg-card">
        {lookupBar}
        <div className="m-3 rounded-md border border-status-danger/30 bg-status-danger/10 p-3 text-xs text-status-danger">
          <div className="mb-1 font-medium">Delivery not found</div>
          <code className="mono break-all text-[11px]">{id}</code>
          {(error as Error)?.message && (
            <div className="mt-1 text-[11px] opacity-90">{(error as Error).message}</div>
          )}
        </div>
      </aside>
    );
  }

  const { delivery, attempts, audit, outbox } = data;
  const sortedAudit = [...audit].sort(
    (a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime(),
  );
  const recipientJSON = fmtJSON(delivery.recipient);
  const templateJSON = fmtJSON(delivery.template);
  const payloadJSON = fmtJSON(delivery.payload);

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
      {lookupBar}

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
            <SectionLabel>Notification</SectionLabel>
            <dl className="mt-2 space-y-1.5 rounded-md border border-border bg-background/40 p-2.5 text-[11px]">
              <MetaRow label="Recipient">
                {recipientJSON ? (
                  <pre className="mono overflow-x-auto whitespace-pre-wrap break-words text-foreground">{recipientJSON}</pre>
                ) : (
                  <span className="text-muted-foreground">—</span>
                )}
              </MetaRow>
              <MetaRow label="Template">
                {templateJSON ? (
                  <pre className="mono overflow-x-auto whitespace-pre-wrap break-words text-foreground">{templateJSON}</pre>
                ) : (
                  <span className="text-muted-foreground">—</span>
                )}
              </MetaRow>
              {payloadJSON && (
                <MetaRow label="Payload">
                  <pre className="mono overflow-x-auto whitespace-pre-wrap break-words text-foreground">{payloadJSON}</pre>
                </MetaRow>
              )}
            </dl>
          </section>

          <section>
            <SectionLabel>Metadata</SectionLabel>
            <dl className="mt-2 space-y-1 rounded-md border border-border bg-background/40 p-2.5 text-[11px]">
              <MetaRow label="Idempotency key">
                <code className="mono break-all">{delivery.idempotency_key}</code>
              </MetaRow>
              {delivery.tenant_id && (
                <MetaRow label="Tenant">
                  <code className="mono">{delivery.tenant_id}</code>
                </MetaRow>
              )}
              {delivery.correlation_id && (
                <MetaRow label="Correlation id">
                  <code className="mono break-all">{delivery.correlation_id}</code>
                </MetaRow>
              )}
              {delivery.causation_id && (
                <MetaRow label="Causation id">
                  <code className="mono break-all">{delivery.causation_id}</code>
                </MetaRow>
              )}
              <MetaRow label="Retries">
                <span className="mono">{delivery.retry_count}</span>
              </MetaRow>
              <MetaRow label="Created">
                <span className="mono text-muted-foreground">{fmt(delivery.created_at)}</span>
              </MetaRow>
              <MetaRow label="Updated">
                <span className="mono text-muted-foreground">{fmt(delivery.updated_at)}</span>
              </MetaRow>
              {delivery.scheduled_at && (
                <MetaRow label="Scheduled at">
                  <span className="mono text-muted-foreground">{fmt(delivery.scheduled_at)}</span>
                </MetaRow>
              )}
            </dl>
          </section>

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
                  {a.payload && Object.keys(a.payload).length > 0 && (
                    <pre className="mt-1 mono text-[10px] text-muted-foreground whitespace-pre-wrap break-words">
                      {JSON.stringify(a.payload, null, 2)}
                    </pre>
                  )}
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
                    {a.success && a.provider_response && Object.keys(a.provider_response).length > 0 && (
                      <pre className="mt-1 mono text-[10px] text-muted-foreground whitespace-pre-wrap break-words">
                        {JSON.stringify(a.provider_response, null, 2)}
                      </pre>
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

function MetaRow({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="grid gap-0.5">
      <dt className="text-[10px] uppercase tracking-wider text-muted-foreground">{label}</dt>
      <dd>{children}</dd>
    </div>
  );
}
