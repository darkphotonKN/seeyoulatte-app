"use client";

import { useEffect } from "react";
import { useUIStore } from "@/stores/ui.store";

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const initializeTheme = useUIStore((state) => state.initializeTheme);

  useEffect(() => {
    // Initialize theme on mount
    initializeTheme();
  }, [initializeTheme]);

  return <>{children}</>;
}