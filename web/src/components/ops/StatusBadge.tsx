import type { DeliveryStatus } from "@/api/types";
import { cn } from "@/lib/utils";

const TONE: Record<DeliveryStatus, { tone: "success" | "warning" | "danger" | "neutral"; label: string }> = {
  succeeded: { tone: "success", label: "succeeded" },
  queued: { tone: "warning", label: "queued" },
  processing: { tone: "warning", label: "processing" },
  retrying: { tone: "warning", label: "retrying" },
  pending: { tone: "warning", label: "pending" },
  dlq: { tone: "danger", label: "dlq" },
  failed: { tone: "danger", label: "failed" },
  rejected: { tone: "danger", label: "rejected" },
  cancelled: { tone: "neutral", label: "cancelled" },
};

export function StatusBadge({ status, className }: { status: DeliveryStatus; className?: string }) {
  const { tone, label } = TONE[status];
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 rounded-md border px-1.5 py-0.5 text-[11px] font-medium uppercase tracking-wide mono",
        tone === "success" && "border-status-success/30 bg-status-success/10 text-status-success",
        tone === "warning" && "border-status-warning/30 bg-status-warning/10 text-status-warning",
        tone === "danger" && "border-status-danger/30 bg-status-danger/10 text-status-danger",
        tone === "neutral" && "border-status-neutral/30 bg-status-neutral/10 text-status-neutral",
        className,
      )}
    >
      <span
        className={cn(
          "h-1.5 w-1.5 rounded-full",
          tone === "success" && "bg-status-success",
          tone === "warning" && "bg-status-warning animate-pulse",
          tone === "danger" && "bg-status-danger",
          tone === "neutral" && "bg-status-neutral",
        )}
      />
      {label}
    </span>
  );
}