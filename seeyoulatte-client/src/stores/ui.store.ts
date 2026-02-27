import { create } from "zustand";
import { persist } from "zustand/middleware";

interface UIState {
  theme: "light" | "dark" | "system";
  sidebarOpen: boolean;
  globalLoading: boolean;
  setTheme: (theme: "light" | "dark" | "system") => void;
  toggleSidebar: () => void;
  setSidebarOpen: (open: boolean) => void;
  setGlobalLoading: (loading: boolean) => void;
  toggleTheme: () => void;
  initializeTheme: () => void;
}

export const useUIStore = create<UIState>()(
  persist(
    (set, get) => ({
      theme: "light", // Default to light mode
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

      toggleTheme: () => {
        const current = get().theme;
        const next = current === "light" ? "dark" : "light";
        get().setTheme(next);
      },

      initializeTheme: () => {
        const theme = get().theme;
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
    }),
    {
      name: "ui-storage",
      partialize: (state) => ({ theme: state.theme }), // Only persist theme
    }
  )
);