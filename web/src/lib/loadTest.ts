import type { Channel, Priority, SendNotificationReq, SendNotificationResp } from "@/api/types";

export const ALL_CHANNELS: Channel[] = ["email", "sms", "push", "webhook"];
export const ALL_PRIORITIES: Priority[] = ["low", "normal", "high"];

export const RECIPIENT_PRESETS = [
  { label: "Happy", address: "user@example.com", tone: "success" as const },
  { label: "Transient", address: "fail-transient@example.com", tone: "warning" as const },
  { label: "Permanent", address: "fail-permanent@example.com", tone: "danger" as const },
  { label: "Circuit", address: "fail-circuit@example.com", tone: "warning" as const },
];

export type ChannelMode = "fixed" | "random" | "round-robin";
export type PriorityMode = "fixed" | "random";

export interface LoadTestConfig {
  address: string;
  burstCount: number;
  duplicateCount: number;
  channelMode: ChannelMode;
  fixedChannel: Channel;
  selectedChannels: Channel[];
  priorityMode: PriorityMode;
  fixedPriority: Priority;
  concurrency: number;
  staggerMs: number;
}

export interface LoadRunResult {
  ok: number;
  fail: number;
  duplicates: number;
  total: number;
  elapsedMs: number;
}

export type LoadScenarioId =
  | "burst"
  | "duplicate-seq"
  | "duplicate-par"
  | "random-channel"
  | "random-priority"
  | "mixed-chaos"
  | "retry-storm"
  | "failure-mix"
  | "staggered"
  | "all-channels";

export const LOAD_SCENARIOS: {
  id: LoadScenarioId;
  label: string;
  description: string;
}[] = [
  {
    id: "burst",
    label: "Parallel burst",
    description: "Unique keys, max concurrency — baseline ingest spike",
  },
  {
    id: "duplicate-seq",
    label: "Duplicate storm (sequential)",
    description: "Same idempotency key, one after another",
  },
  {
    id: "duplicate-par",
    label: "Duplicate storm (parallel)",
    description: "Same idempotency key, fired concurrently — idempotency race",
  },
  {
    id: "random-channel",
    label: "Random channel burst",
    description: "Each request picks a random channel from selection",
  },
  {
    id: "random-priority",
    label: "Random priority burst",
    description: "Each request picks low / normal / high at random",
  },
  {
    id: "mixed-chaos",
    label: "Mixed chaos",
    description: "Random channel + priority per request",
  },
  {
    id: "retry-storm",
    label: "Retry storm",
    description: "All fail-transient recipients — worker retry pressure",
  },
  {
    id: "failure-mix",
    label: "Failure mix",
    description: "Random happy / transient / permanent recipients",
  },
  {
    id: "staggered",
    label: "Staggered burst",
    description: "Sequential submits with configurable delay",
  },
  {
    id: "all-channels",
    label: "All-channels sweep",
    description: "Round-robin across selected channels",
  },
];

export function genLoadKey(prefix = "load") {
  return `${prefix}-${Math.random().toString(36).slice(2, 8)}-${Date.now().toString(36).slice(-4)}`;
}

export function pickRandom<T>(items: readonly T[]): T {
  return items[Math.floor(Math.random() * items.length)]!;
}

export function resolveChannel(config: LoadTestConfig, index: number): Channel {
  const pool =
    config.channelMode === "fixed"
      ? [config.fixedChannel]
      : config.selectedChannels.length > 0
        ? config.selectedChannels
        : ALL_CHANNELS;

  if (config.channelMode === "round-robin" || config.channelMode === "random") {
    if (config.channelMode === "round-robin") {
      return pool[index % pool.length]!;
    }
    return pickRandom(pool);
  }
  return config.fixedChannel;
}

export function resolvePriority(config: LoadTestConfig): Priority {
  if (config.priorityMode === "random") {
    return pickRandom(ALL_PRIORITIES);
  }
  return config.fixedPriority;
}

export function buildRequest(
  config: LoadTestConfig,
  index: number,
  overrides?: Partial<SendNotificationReq>,
): SendNotificationReq {
  return {
    schema_version: "1.0",
    idempotency_key: genLoadKey(),
    channel: resolveChannel(config, index),
    priority: resolvePriority(config),
    recipient: { address: config.address },
    template: { id: "welcome", data: {} },
    ...overrides,
  };
}

const FAILURE_MIX_ADDRESSES = [
  "user@example.com",
  "fail-transient@example.com",
  "fail-permanent@example.com",
];

export function buildScenarioRequest(
  scenario: LoadScenarioId,
  config: LoadTestConfig,
  index: number,
  sharedKey?: string,
): SendNotificationReq {
  switch (scenario) {
    case "retry-storm":
      return buildRequest(config, index, {
        channel: config.channelMode === "fixed" ? config.fixedChannel : resolveChannel(config, index),
        priority: config.priorityMode === "fixed" ? config.fixedPriority : resolvePriority(config),
        recipient: { address: "fail-transient@example.com" },
      });
    case "failure-mix":
      return buildRequest(config, index, {
        recipient: { address: pickRandom(FAILURE_MIX_ADDRESSES) },
      });
    case "random-channel":
      return buildRequest(
        { ...config, channelMode: "random", priorityMode: config.priorityMode },
        index,
      );
    case "random-priority":
      return buildRequest(
        { ...config, channelMode: config.channelMode, priorityMode: "random" },
        index,
      );
    case "mixed-chaos":
      return buildRequest(
        { ...config, channelMode: "random", priorityMode: "random" },
        index,
      );
    case "all-channels":
      return buildRequest({ ...config, channelMode: "round-robin" }, index);
    case "duplicate-seq":
    case "duplicate-par":
      return buildRequest(config, index, { idempotency_key: sharedKey ?? genLoadKey("dup") });
    default:
      return buildRequest(config, index);
  }
}

export function scenarioCount(scenario: LoadScenarioId, config: LoadTestConfig): number {
  if (scenario === "duplicate-seq" || scenario === "duplicate-par") {
    return config.duplicateCount;
  }
  return config.burstCount;
}

async function sendOne(
  send: (req: SendNotificationReq) => Promise<SendNotificationResp>,
  req: SendNotificationReq,
): Promise<"ok" | "fail" | "duplicate"> {
  try {
    const res = await send(req);
    return res.duplicate ? "duplicate" : "ok";
  } catch {
    return "fail";
  }
}

export async function runParallel(
  count: number,
  concurrency: number,
  build: (index: number) => SendNotificationReq,
  send: (req: SendNotificationReq) => Promise<SendNotificationResp>,
  onProgress?: (result: LoadRunResult) => void,
): Promise<LoadRunResult> {
  const started = performance.now();
  const result: LoadRunResult = { ok: 0, fail: 0, duplicates: 0, total: count, elapsedMs: 0 };
  const limit = Math.max(1, Math.min(concurrency, count));
  let next = 0;

  async function worker() {
    while (true) {
      const i = next++;
      if (i >= count) break;
      const outcome = await sendOne(send, build(i));
      if (outcome === "ok") result.ok++;
      else if (outcome === "duplicate") result.duplicates++;
      else result.fail++;
      onProgress?.({ ...result, elapsedMs: performance.now() - started });
    }
  }

  await Promise.all(Array.from({ length: limit }, () => worker()));
  result.elapsedMs = performance.now() - started;
  return result;
}

export async function runSequential(
  count: number,
  staggerMs: number,
  build: (index: number) => SendNotificationReq,
  send: (req: SendNotificationReq) => Promise<SendNotificationResp>,
  onProgress?: (result: LoadRunResult) => void,
): Promise<LoadRunResult> {
  const started = performance.now();
  const result: LoadRunResult = { ok: 0, fail: 0, duplicates: 0, total: count, elapsedMs: 0 };

  for (let i = 0; i < count; i++) {
    const outcome = await sendOne(send, build(i));
    if (outcome === "ok") result.ok++;
    else if (outcome === "duplicate") result.duplicates++;
    else result.fail++;
    onProgress?.({ ...result, elapsedMs: performance.now() - started });
    if (staggerMs > 0 && i < count - 1) {
      await sleep(staggerMs);
    }
  }

  result.elapsedMs = performance.now() - started;
  return result;
}

function sleep(ms: number) {
  return new Promise<void>((resolve) => setTimeout(resolve, ms));
}

export function formatRunSummary(result: LoadRunResult): string {
  const parts = [`${result.ok} accepted`, `${result.fail} failed`];
  if (result.duplicates > 0) {
    parts.push(`${result.duplicates} duplicates`);
  }
  parts.push(`${result.total} total`, `${Math.round(result.elapsedMs)} ms`);
  return parts.join(", ");
}

export function defaultLoadConfig(): LoadTestConfig {
  return {
    address: "user@example.com",
    burstCount: 20,
    duplicateCount: 25,
    channelMode: "fixed",
    fixedChannel: "email",
    selectedChannels: [...ALL_CHANNELS],
    priorityMode: "fixed",
    fixedPriority: "normal",
    concurrency: 10,
    staggerMs: 50,
  };
}
