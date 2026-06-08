import {
  AlertTriangle,
  ChevronDown,
  Loader2,
  Shuffle,
  Waves,
  Zap,
} from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import type { Channel, Priority } from "@/api/types";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useSendNotification } from "@/hooks/useDashboard";
import {
  ALL_CHANNELS,
  ALL_PRIORITIES,
  defaultLoadConfig,
  formatRunSummary,
  LOAD_SCENARIOS,
  RECIPIENT_PRESETS,
  runParallel,
  runSequential,
  scenarioCount,
  buildScenarioRequest,
  type LoadRunResult,
  type LoadScenarioId,
  type LoadTestConfig,
} from "@/lib/loadTest";
import { cn } from "@/lib/utils";

function scenarioIcon(id: LoadScenarioId) {
  if (id.startsWith("duplicate")) return Shuffle;
  if (id === "staggered") return Waves;
  if (id === "retry-storm" || id === "failure-mix") return AlertTriangle;
  return Zap;
}

export function LoadTestPanel() {
  const [config, setConfig] = useState<LoadTestConfig>(defaultLoadConfig);
  const [advancedOpen, setAdvancedOpen] = useState(false);
  const [running, setRunning] = useState<LoadScenarioId | null>(null);
  const [progress, setProgress] = useState<LoadRunResult | null>(null);

  const { mutateAsync } = useSendNotification();

  function patch(partial: Partial<LoadTestConfig>) {
    setConfig((c) => ({ ...c, ...partial }));
  }

  function toggleChannel(channel: Channel, checked: boolean) {
    setConfig((c) => {
      const next = checked
        ? [...new Set([...c.selectedChannels, channel])]
        : c.selectedChannels.filter((ch) => ch !== channel);
      return { ...c, selectedChannels: next.length > 0 ? next : [channel] };
    });
  }

  async function runScenario(scenario: LoadScenarioId) {
    const count = scenarioCount(scenario, config);
    if (count < 1) {
      toast.error("Invalid count", { description: "Count must be at least 1." });
      return;
    }

    setRunning(scenario);
    setProgress(null);
    const sharedKey = scenario.startsWith("duplicate") ? `dup-storm-${Date.now()}` : undefined;
    const meta = LOAD_SCENARIOS.find((s) => s.id === scenario)!;

    try {
      const onProgress = (r: LoadRunResult) => setProgress({ ...r });

      const result =
        scenario === "duplicate-seq" || scenario === "staggered"
          ? await runSequential(
              count,
              scenario === "staggered" ? config.staggerMs : 0,
              (i) => buildScenarioRequest(scenario, config, i, sharedKey),
              mutateAsync,
              onProgress,
            )
          : await runParallel(
              count,
              config.concurrency,
              (i) => buildScenarioRequest(scenario, config, i, sharedKey),
              mutateAsync,
              onProgress,
            );

      const tone = result.fail > 0 ? "message" : "success";
      toast[tone](`${meta.label} complete`, {
        description: formatRunSummary(result),
      });
    } finally {
      setRunning(null);
      setProgress(null);
    }
  }

  const activeScenario = running ? LOAD_SCENARIOS.find((s) => s.id === running) : null;

  return (
    <div className="space-y-3 rounded-lg border border-border bg-card p-3">
      <div className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
        Load test panel
      </div>

      <div className="flex flex-wrap gap-1.5">
        {RECIPIENT_PRESETS.map((p) => (
          <button
            key={p.label}
            type="button"
            onClick={() => patch({ address: p.address })}
            className={cn(
              "rounded-md border px-2 py-1 text-[11px] transition-colors",
              config.address === p.address
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
        <Label className="text-[11px] text-muted-foreground">Recipient</Label>
        <Input
          value={config.address}
          onChange={(e) => patch({ address: e.target.value })}
          className="h-8 text-xs mono"
        />
      </div>

      <div className="grid grid-cols-2 gap-2">
        <div className="space-y-1.5">
          <Label className="text-[11px] text-muted-foreground">Burst count</Label>
          <Input
            type="number"
            min={1}
            max={500}
            value={config.burstCount}
            onChange={(e) => patch({ burstCount: Math.max(1, Number(e.target.value) || 1) })}
            className="h-8 text-xs mono"
          />
        </div>
        <div className="space-y-1.5">
          <Label className="text-[11px] text-muted-foreground">Duplicate count</Label>
          <Input
            type="number"
            min={1}
            max={500}
            value={config.duplicateCount}
            onChange={(e) => patch({ duplicateCount: Math.max(1, Number(e.target.value) || 1) })}
            className="h-8 text-xs mono"
          />
        </div>
      </div>

      <Collapsible open={advancedOpen} onOpenChange={setAdvancedOpen}>
        <CollapsibleTrigger asChild>
          <button
            type="button"
            className="flex w-full items-center justify-between rounded-md border border-border px-2 py-1.5 text-[11px] text-muted-foreground hover:text-foreground"
          >
            <span>Advanced configuration</span>
            <ChevronDown
              className={cn("h-3.5 w-3.5 transition-transform", advancedOpen && "rotate-180")}
            />
          </button>
        </CollapsibleTrigger>
        <CollapsibleContent className="mt-2 space-y-2">
          <div className="space-y-1.5">
            <Label className="text-[11px] text-muted-foreground">Channel mode</Label>
            <Select
              value={config.channelMode}
              onValueChange={(v) =>
                patch({ channelMode: v as LoadTestConfig["channelMode"] })
              }
            >
              <SelectTrigger className="h-8 text-xs mono">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="fixed">Fixed channel</SelectItem>
                <SelectItem value="random">Random from selection</SelectItem>
                <SelectItem value="round-robin">Round-robin sweep</SelectItem>
              </SelectContent>
            </Select>
          </div>

          {config.channelMode === "fixed" ? (
            <div className="space-y-1.5">
              <Label className="text-[11px] text-muted-foreground">Channel</Label>
              <Select
                value={config.fixedChannel}
                onValueChange={(v) => patch({ fixedChannel: v as Channel })}
              >
                <SelectTrigger className="h-8 text-xs mono">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {ALL_CHANNELS.map((ch) => (
                    <SelectItem key={ch} value={ch}>
                      {ch}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          ) : (
            <div className="space-y-1.5">
              <Label className="text-[11px] text-muted-foreground">Channels in pool</Label>
              <div className="grid grid-cols-2 gap-2">
                {ALL_CHANNELS.map((ch) => (
                  <label
                    key={ch}
                    className="flex cursor-pointer items-center gap-2 text-[11px] mono"
                  >
                    <Checkbox
                      checked={config.selectedChannels.includes(ch)}
                      onCheckedChange={(checked) => toggleChannel(ch, checked === true)}
                    />
                    {ch}
                  </label>
                ))}
              </div>
            </div>
          )}

          <div className="grid grid-cols-2 gap-2">
            <div className="space-y-1.5">
              <Label className="text-[11px] text-muted-foreground">Priority mode</Label>
              <Select
                value={config.priorityMode}
                onValueChange={(v) =>
                  patch({ priorityMode: v as LoadTestConfig["priorityMode"] })
                }
              >
                <SelectTrigger className="h-8 text-xs mono">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="fixed">Fixed</SelectItem>
                  <SelectItem value="random">Random</SelectItem>
                </SelectContent>
              </Select>
            </div>
            {config.priorityMode === "fixed" && (
              <div className="space-y-1.5">
                <Label className="text-[11px] text-muted-foreground">Priority</Label>
                <Select
                  value={config.fixedPriority}
                  onValueChange={(v) => patch({ fixedPriority: v as Priority })}
                >
                  <SelectTrigger className="h-8 text-xs mono">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {ALL_PRIORITIES.map((p) => (
                      <SelectItem key={p} value={p}>
                        {p}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            )}
          </div>

          <div className="grid grid-cols-2 gap-2">
            <div className="space-y-1.5">
              <Label className="text-[11px] text-muted-foreground">Concurrency</Label>
              <Input
                type="number"
                min={1}
                max={100}
                value={config.concurrency}
                onChange={(e) =>
                  patch({ concurrency: Math.max(1, Math.min(100, Number(e.target.value) || 1)) })
                }
                className="h-8 text-xs mono"
              />
            </div>
            <div className="space-y-1.5">
              <Label className="text-[11px] text-muted-foreground">Stagger (ms)</Label>
              <Input
                type="number"
                min={0}
                max={5000}
                value={config.staggerMs}
                onChange={(e) =>
                  patch({ staggerMs: Math.max(0, Number(e.target.value) || 0) })
                }
                className="h-8 text-xs mono"
              />
            </div>
          </div>
        </CollapsibleContent>
      </Collapsible>

      {running && progress && (
        <div className="rounded-md border border-border bg-muted/40 px-2 py-1.5 text-[10px] mono text-muted-foreground">
          {activeScenario?.label}: {progress.ok + progress.duplicates}/{progress.total} (
          {progress.fail} failed) · {Math.round(progress.elapsedMs)} ms
        </div>
      )}

      <div className="flex max-h-[220px] flex-col gap-1.5 overflow-y-auto pr-0.5">
        {LOAD_SCENARIOS.map((scenario) => {
          const Icon = scenarioIcon(scenario.id);
          const count = scenarioCount(scenario.id, config);
          const isRunning = running === scenario.id;
          return (
            <Button
              key={scenario.id}
              type="button"
              variant={scenario.id.startsWith("duplicate") ? "outline" : "secondary"}
              disabled={running !== null}
              className="h-auto min-h-8 justify-start py-1.5 text-left text-xs"
              title={scenario.description}
              onClick={() => runScenario(scenario.id)}
            >
              {isRunning ? (
                <Loader2 className="mr-1.5 h-3.5 w-3.5 shrink-0 animate-spin" />
              ) : (
                <Icon className="mr-1.5 h-3.5 w-3.5 shrink-0" />
              )}
              <span className="flex min-w-0 flex-col">
                <span>
                  {scenario.label} ({count})
                </span>
                <span className="truncate text-[10px] font-normal text-muted-foreground">
                  {scenario.description}
                </span>
              </span>
            </Button>
          );
        })}
      </div>
    </div>
  );
}
