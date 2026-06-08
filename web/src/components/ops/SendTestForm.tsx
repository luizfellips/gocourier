import { Dices, Send } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import type { Channel } from "@/api/types";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useSendNotification } from "@/hooks/useDashboard";
import { useAppStore } from "@/stores/appStore";
import { cn } from "@/lib/utils";

const PRESETS = [
  { label: "Happy path", addr: "user@example.com", tone: "success" },
  { label: "Permanent fail", addr: "fail-permanent@example.com", tone: "danger" },
  { label: "Transient fail", addr: "fail-transient@example.com", tone: "warning" },
] as const;

function genKey() {
  return `test-${Math.random().toString(36).slice(2, 8)}-${Date.now().toString(36).slice(-4)}`;
}

export function SendTestForm() {
  const [channel, setChannel] = useState<Channel>("email");
  const [address, setAddress] = useState("user@example.com");
  const [key, setKey] = useState(genKey());
  const setSelected = useAppStore((s) => s.setSelectedDeliveryId);

  const { mutate, isPending } = useSendNotification();

  function submit(e: React.FormEvent) {
    e.preventDefault();
    mutate(
      {
        schema_version: "1.0",
        idempotency_key: key,
        channel,
        priority: "normal",
        recipient: { address },
        template: { id: "welcome", data: {} },
      },
      {
        onSuccess: (res) => {
          if (res.duplicate) {
            toast.message("Duplicate idempotency key", {
              description: `Returned existing delivery ${res.delivery_id.slice(0, 8)}…`,
            });
          } else {
            toast.success("Notification queued", {
              description: `delivery_id ${res.delivery_id.slice(0, 8)}…`,
            });
          }
          setSelected(res.delivery_id);
          setKey(genKey());
        },
        onError: (err) => toast.error("Send failed", { description: (err as Error).message }),
      },
    );
  }

  return (
    <form onSubmit={submit} className="space-y-3 rounded-lg border border-border bg-card p-3">
      <div className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
        Send test notification
      </div>

      <div className="flex flex-wrap gap-1.5">
        {PRESETS.map((p) => (
          <button
            key={p.label}
            type="button"
            onClick={() => {
              setAddress(p.addr);
              setChannel("email");
            }}
            className={cn(
              "rounded-md border px-2 py-1 text-[11px] transition-colors",
              address === p.addr
                ? p.tone === "success"
                  ? "border-status-success/40 bg-status-success/10 text-status-success"
                  : p.tone === "danger"
                  ? "border-status-danger/40 bg-status-danger/10 text-status-danger"
                  : "border-status-warning/40 bg-status-warning/10 text-status-warning"
                : "border-border bg-background text-muted-foreground hover:text-foreground",
            )}
          >
            {p.label}
          </button>
        ))}
      </div>

      <div className="space-y-1.5">
        <Label className="text-[11px] text-muted-foreground">Channel</Label>
        <Select value={channel} onValueChange={(v) => setChannel(v as Channel)}>
          <SelectTrigger className="h-8 text-xs mono">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="email">email</SelectItem>
            <SelectItem value="sms">sms</SelectItem>
            <SelectItem value="push">push</SelectItem>
            <SelectItem value="webhook">webhook</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <div className="space-y-1.5">
        <Label className="text-[11px] text-muted-foreground">Recipient</Label>
        <Input
          value={address}
          onChange={(e) => setAddress(e.target.value)}
          className="h-8 text-xs mono"
          placeholder="user@example.com"
        />
      </div>

      <div className="space-y-1.5">
        <Label className="text-[11px] text-muted-foreground">Idempotency key</Label>
        <div className="flex gap-1.5">
          <Input value={key} onChange={(e) => setKey(e.target.value)} className="h-8 text-xs mono" />
          <Button
            type="button"
            variant="outline"
            size="icon"
            className="h-8 w-8 shrink-0"
            onClick={() => setKey(genKey())}
            title="Regenerate"
          >
            <Dices className="h-3.5 w-3.5" />
          </Button>
        </div>
      </div>

      <Button type="submit" disabled={isPending} className="w-full h-8 text-xs">
        <Send className="mr-1.5 h-3.5 w-3.5" />
        {isPending ? "Sending…" : "POST /v1/notifications"}
      </Button>
    </form>
  );
}