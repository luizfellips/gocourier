import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  getDelivery,
  getHealth,
  getSummary,
  replayDelivery,
  sendNotification,
} from "@/api/client";
import type { SendNotificationReq } from "@/api/types";
import { useAppStore } from "@/stores/appStore";

export function useHealth() {
  const autoRefresh = useAppStore((s) => s.autoRefresh);
  return useQuery({
    queryKey: ["health"],
    queryFn: getHealth,
    refetchInterval: autoRefresh ? 5000 : false,
    retry: false,
  });
}

export function useSummary() {
  const autoRefresh = useAppStore((s) => s.autoRefresh);
  return useQuery({
    queryKey: ["summary"],
    queryFn: () => getSummary(30),
    refetchInterval: autoRefresh ? 2000 : false,
  });
}

export function useDeliveryDetail(id: string | null) {
  const autoRefresh = useAppStore((s) => s.autoRefresh);
  return useQuery({
    queryKey: ["delivery", id],
    queryFn: () => getDelivery(id!),
    enabled: !!id,
    refetchInterval: autoRefresh && id ? 2000 : false,
  });
}

export function useSendNotification() {
  const qc = useQueryClient();
  const apiKey = useAppStore((s) => s.apiKey);
  return useMutation({
    mutationFn: (req: SendNotificationReq) => sendNotification(req, apiKey),
    onSuccess: (res) => {
      qc.invalidateQueries({ queryKey: ["summary"] });
      useAppStore.getState().setSelectedDeliveryId(res.delivery_id);
    },
  });
}

export function useReplay() {
  const qc = useQueryClient();
  const apiKey = useAppStore((s) => s.apiKey);
  return useMutation({
    mutationFn: (id: string) => replayDelivery(id, apiKey),
    onSuccess: (_, id) => {
      qc.invalidateQueries({ queryKey: ["summary"] });
      qc.invalidateQueries({ queryKey: ["delivery", id] });
    },
  });
}