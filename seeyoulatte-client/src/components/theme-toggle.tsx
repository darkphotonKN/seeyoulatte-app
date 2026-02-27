"use client";

import { useEffect, useState } from "react";
import { Moon, Sun } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useUIStore } from "@/stores/ui.store";

export function ThemeToggle() {
  const [mounted, setMounted] = useState(false);
  const theme = useUIStore((state) => state.theme);
  const toggleTheme = useUIStore((state) => state.toggleTheme);

  // Prevent hydration mismatch
  useEffect(() => {
    setMounted(true);
  }, []);

  if (!mounted) {
    return (
      <Button
        variant="ghost"
        size="icon"
        className="h-9 w-9 rounded-md"
        aria-label="Toggle theme"
      >
        <div className="h-5 w-5" />
      </Button>
    );
  }

  const isDark = theme === "dark" ||
    (theme === "system" && typeof window !== "undefined" &&
     window.matchMedia("(prefers-color-scheme: dark)").matches);

  return (
    <Button
      variant="ghost"
      size="icon"
      onClick={toggleTheme}
      className="h-9 w-9 rounded-md hover:bg-accent/10"
      aria-label="Toggle theme"
    >
      {isDark ? (
        <Sun className="h-5 w-5 text-coffee-golden transition-all" />
      ) : (
        <Moon className="h-5 w-5 text-coffee-espresso transition-all" />
      )}
    </Button>
  );
}