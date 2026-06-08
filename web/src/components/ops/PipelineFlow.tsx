import { Database, Inbox, Mail, Network, Send, Server } from "lucide-react";
import { Fragment } from "react";
import type { DeliveryStatus } from "@/api/types";
import { cn } from "@/lib/utils";

type StepId = "ingest" | "pg" | "outbox" | "nats" | "worker" | "provider";

const STEPS: { id: StepId; label: string; icon: typeof Inbox }[] = [
  { id: "ingest", label: "Ingest API", icon: Inbox },
  { id: "pg", label: "PostgreSQL", icon: Database },
  { id: "outbox", label: "Outbox", icon: Send },
  { id: "nats", label: "NATS JetStream", icon: Network },
  { id: "worker", label: "Worker", icon: Server },
  { id: "provider", label: "Provider", icon: Mail },
];

function activeStepsFor(status: DeliveryStatus | undefined): Set<StepId> {
  if (!status) return new Set();
  switch (status) {
    case "pending":
      return new Set(["ingest"]);
    case "queued":
      return new Set(["ingest", "pg", "outbox"]);
    case "processing":
    case "retrying":
      return new Set(["ingest", "pg", "outbox", "nats", "worker"]);
    case "succeeded":
      return new Set(["ingest", "pg", "outbox", "nats", "worker", "provider"]);
    case "dlq":
    case "failed":
    case "rejected":
      return new Set(["ingest", "pg", "outbox", "nats", "worker"]);
    case "cancelled":
      return new Set(["ingest", "pg"]);
  }
}

function tone(status: DeliveryStatus | undefined): "success" | "warning" | "danger" | "neutral" {
  if (!status) return "neutral";
  if (status === "succeeded") return "success";
  if (status === "dlq" || status === "failed" || status === "rejected") return "danger";
  if (status === "cancelled") return "neutral";
  return "warning";
}

export function PipelineFlow({ selectedStatus }: { selectedStatus?: DeliveryStatus }) {
  const active = activeStepsFor(selectedStatus);
  const t = tone(selectedStatus);
  const head = STEPS[STEPS.length - 1].id;
  const stoppedAt = selectedStatus && t !== "success" && t !== "neutral" ? [...active].pop() : null;

  return (
    <div className="rounded-lg border border-border bg-card">
      <div className="flex items-center justify-between gap-4 border-b border-border px-4 py-2">
        <div className="text-xs font-medium uppercase tracking-wider text-muted-foreground">Pipeline</div>
        {selectedStatus ? (
          <div className="text-[11px] text-muted-foreground mono">
            tracking selected delivery → <span className="text-foreground">{selectedStatus}</span>
          </div>
        ) : (
          <div className="text-[11px] text-muted-foreground">Select a delivery to trace its position</div>
        )}
      </div>
      <div className="flex items-stretch gap-1 overflow-x-auto px-3 py-4">
        {STEPS.map((step, idx) => {
          const Icon = step.icon;
          const isActive = active.has(step.id);
          const isHead = stoppedAt === step.id || (t === "success" && step.id === head);
          return (
            <Fragment key={step.id}>
              <div className="flex min-w-[110px] flex-1 flex-col items-center gap-1.5">
                <div
                  className={cn(
                    "grid h-10 w-10 place-items-center rounded-md border transition-colors",
                    !isActive && "border-border bg-background text-muted-foreground",
                    isActive && t === "warning" && "border-status-warning/40 bg-status-warning/10 text-status-warning",
                    isActive && t === "success" && "border-status-success/40 bg-status-success/10 text-status-success",
                    isActive && t === "danger" && "border-status-danger/40 bg-status-danger/10 text-status-danger",
                    isActive && t === "neutral" && "border-border bg-secondary text-foreground",
                    isHead && "ring-2 ring-offset-2 ring-offset-card",
                    isHead && t === "warning" && "ring-status-warning/50",
                    isHead && t === "success" && "ring-status-success/50",
                    isHead && t === "danger" && "ring-status-danger/50",
                  )}
                >
                  <Icon className="h-4 w-4" />
                </div>
                <div className={cn("text-[11px] mono", isActive ? "text-foreground" : "text-muted-foreground")}>
                  {step.label}
                </div>
              </div>
              {idx < STEPS.length - 1 && (
                <div className="flex flex-1 items-center pt-5">
                  <div
                    className={cn(
                      "h-px w-full",
                      active.has(step.id) && active.has(STEPS[idx + 1].id)
                        ? t === "danger"
                          ? "bg-status-danger/50"
                          : t === "success"
                          ? "bg-status-success/50"
                          : "bg-status-warning/50"
                        : "bg-border",
                    )}
                  />
                </div>
              )}
            </Fragment>
          );
        })}
      </div>
    </div>
  );
}