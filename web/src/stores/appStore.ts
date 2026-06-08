import { create } from "zustand";
import { persist } from "zustand/middleware";

interface AppState {
  apiKey: string;
  autoRefresh: boolean;
  selectedDeliveryId: string | null;
  setApiKey: (k: string) => void;
  setAutoRefresh: (v: boolean) => void;
  setSelectedDeliveryId: (id: string | null) => void;
}

export const useAppStore = create<AppState>()(
  persist(
    (set) => ({
      apiKey: "dev-api-key",
      autoRefresh: true,
      selectedDeliveryId: null,
      setApiKey: (apiKey) => set({ apiKey }),
      setAutoRefresh: (autoRefresh) => set({ autoRefresh }),
      setSelectedDeliveryId: (selectedDeliveryId) => set({ selectedDeliveryId }),
    }),
    {
      name: "ep-ops-store",
      partialize: (s) => ({ apiKey: s.apiKey, autoRefresh: s.autoRefresh }),
    },
  ),
);