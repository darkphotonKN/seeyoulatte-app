import { create } from "zustand";

interface UIState {
  theme: "light" | "dark" | "system";
  sidebarOpen: boolean;
  globalLoading: boolean;
  setTheme: (theme: "light" | "dark" | "system") => void;
  toggleSidebar: () => void;
  setSidebarOpen: (open: boolean) => void;
  setGlobalLoading: (loading: boolean) => void;
}

export const useUIStore = create<UIState>((set) => ({
  theme: "system",
  sidebarOpen: true,
  globalLoading: false,

  setTheme: (theme) => {
    set({ theme });
    // Apply theme to document
    if (typeof window !== "undefined") {
      const root = window.document.documentElement;
      root.classList.remove("light", "dark");

      if (theme === "system") {
        const systemTheme = window.matchMedia("(prefers-color-scheme: dark)").matches
          ? "dark"
          : "light";
        root.classList.add(systemTheme);
      } else {
        root.classList.add(theme);
      }
    }
  },

  toggleSidebar: () => {
    set((state) => ({ sidebarOpen: !state.sidebarOpen }));
  },

  setSidebarOpen: (open) => {
    set({ sidebarOpen: open });
  },

  setGlobalLoading: (loading) => {
    set({ globalLoading: loading });
  },
}));