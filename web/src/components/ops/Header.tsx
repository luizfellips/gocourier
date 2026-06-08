import { Activity, KeyRound, RefreshCw } from "lucide-react";
import { useEffect, useState } from "react";
import { Input } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";
import { useHealth, useSummary } from "@/hooks/useDashboard";
import { useAppStore } from "@/stores/appStore";
import { cn } from "@/lib/utils";

function useNow() {
  const [now, setNow] = useState(Date.now());
  useEffect(() => {
    const i = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(i);
  }, []);
  return now;
}

function relative(date: number, now: number) {
  const s = Math.max(0, Math.floor((now - date) / 1000));
  if (s < 2) return "just now";
  if (s < 60) return `${s}s ago`;
  const m = Math.floor(s / 60);
  if (m < 60) return `${m}m ago`;
  return `${Math.floor(m / 60)}h ago`;
}

export function Header() {
  const apiKey = useAppStore((s) => s.apiKey);
  const setApiKey = useAppStore((s) => s.setApiKey);
  const autoRefresh = useAppStore((s) => s.autoRefresh);
  const setAutoRefresh = useAppStore((s) => s.setAutoRefresh);

  const health = useHealth();
  const summary = useSummary();
  const now = useNow();
  const lastUpdated = summary.dataUpdatedAt;

  const healthy = health.data?.status === "ok" && !health.isError;

  return (
    <header className="sticky top-0 z-20 border-b border-border bg-background/80 backdrop-blur">
      <div className="mx-auto flex max-w-[1600px] flex-wrap items-center gap-x-6 gap-y-3 px-4 py-3 sm:px-6">
        <div className="flex items-center gap-3">
          <div className="grid h-8 w-8 place-items-center rounded-md border border-border bg-card">
            <Activity className="h-4 w-4 text-status-info" />
          </div>
          <div className="leading-tight">
            <div className="text-sm font-semibold tracking-tight">Gocourier</div>
            <div className="text-[11px] text-muted-foreground">Notification pipeline — live ops</div>
          </div>
        </div>

        <div className="flex items-center gap-2">
          <span
            className={cn(
              "inline-flex items-center gap-1.5 rounded-full border px-2 py-0.5 text-[11px] mono",
              healthy
                ? "border-status-success/30 bg-status-success/10 text-status-success"
                : "border-status-danger/30 bg-status-danger/10 text-status-danger",
            )}
          >
            <span className={cn("h-1.5 w-1.5 rounded-full", healthy ? "bg-status-success" : "bg-status-danger animate-pulse")} />
            api {healthy ? "ok" : "down"}
          </span>
        </div>

        <div className="ml-auto flex flex-wrap items-center gap-4">
          <div className="flex items-center gap-2 text-[11px] text-muted-foreground">
            <RefreshCw className={cn("h-3.5 w-3.5", autoRefresh && summary.isFetching && "animate-spin")} />
            <span className="mono">{lastUpdated ? relative(lastUpdated, now) : "—"}</span>
            <Switch checked={autoRefresh} onCheckedChange={setAutoRefresh} aria-label="auto refresh" />
            <span>auto&nbsp;2s</span>
          </div>

          <div className="flex items-center gap-2">
            <KeyRound className="h-3.5 w-3.5 text-muted-foreground" />
            <Input
              type="password"
              autoComplete="off"
              spellCheck={false}
              value={apiKey}
              onChange={(e) => setApiKey(e.target.value)}
              placeholder="X-API-Key"
              className="h-8 w-44 mono text-xs"
            />
          </div>
        </div>
      </div>
    </header>
  );
}