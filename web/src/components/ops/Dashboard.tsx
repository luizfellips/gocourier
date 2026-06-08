import { AlertTriangle } from "lucide-react";
import { useMemo } from "react";
import { useSummary, useDeliveryDetail, useHealth } from "@/hooks/useDashboard";
import { useAppStore } from "@/stores/appStore";
import { ActivityFeed } from "./ActivityFeed";
import { DeliveriesTable } from "./DeliveriesTable";
import { DeliveryDetail } from "./DeliveryDetail";
import { Header } from "./Header";
import { PipelineFlow } from "./PipelineFlow";
import { LoadTestPanel } from "./LoadTestPanel";
import { SendTestForm } from "./SendTestForm";
import { StatsBar } from "./StatsBar";

export function Dashboard() {
  const summaryQ = useSummary();
  const healthQ = useHealth();
  const selectedId = useAppStore((s) => s.selectedDeliveryId);
  const detailQ = useDeliveryDetail(selectedId);

  const selectedStatus = detailQ.data?.delivery.status;
  const rows = useMemo(() => summaryQ.data?.deliveries ?? [], [summaryQ.data]);
  const events = useMemo(() => summaryQ.data?.recent_activity ?? [], [summaryQ.data]);

  const apiDown = healthQ.isError;

  return (
    <div className="flex min-h-screen flex-col">
      <Header />
      <main className="mx-auto w-full max-w-[1600px] flex-1 px-4 py-4 sm:px-6">
        {apiDown && (
          <div className="mb-3 flex items-center gap-2 rounded-md border border-status-danger/40 bg-status-danger/10 px-3 py-2 text-xs text-status-danger">
            <AlertTriangle className="h-3.5 w-3.5" />
            Can't reach API — is <code className="mono">docker compose up</code>?
          </div>
        )}

        <div className="space-y-4">
          <PipelineFlow selectedStatus={selectedStatus} />
          <StatsBar summary={summaryQ.data} />

          <div className="grid gap-4 lg:grid-cols-[300px_minmax(0,1fr)_380px]">
            <div className="flex flex-col gap-4">
              <SendTestForm />
              <LoadTestPanel />
              <ActivityFeed events={events} />
            </div>
            <div className="min-h-[520px]">
              <DeliveriesTable rows={rows} />
            </div>
            <div className="min-h-[520px]">
              <DeliveryDetail />
            </div>
          </div>
        </div>

        <footer className="mt-6 text-center text-[10px] text-muted-foreground mono">
          connected to Go API · postgres + nats jetstream + worker
        </footer>
      </main>
    </div>
  );
}