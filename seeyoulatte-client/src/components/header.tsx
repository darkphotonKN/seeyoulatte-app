"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { Coffee } from "lucide-react";
import { ThemeToggle } from "./theme-toggle";
import { Button } from "@/components/ui/button";
import { useAuthStore } from "@/stores/auth.store";
import { useRouter } from "next/navigation";

export function Header() {
  const [mounted, setMounted] = useState(false);
  const user = useAuthStore((state) => state.user);
  const logout = useAuthStore((state) => state.logout);
  const router = useRouter();

  useEffect(() => {
    setMounted(true);
  }, []);

  const handleLogout = () => {
    logout();
    router.push("/");
  };

  return (
    <header className="sticky top-0 z-50 w-full border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
      <div className="container mx-auto flex h-16 items-center px-4">
        <div className="flex flex-1 items-center justify-between">
          {/* Logo and Navigation */}
          <div className="flex items-center space-x-6">
            <Link href="/" className="flex items-center space-x-2 hover:opacity-90 transition-opacity">
              <Coffee className="h-6 w-6 text-primary" />
              <span className="font-serif text-xl font-semibold text-foreground">SeeYouLatte</span>
            </Link>

            <nav className="hidden md:flex items-center space-x-6">
              <Link href="/listings" className="text-sm font-medium text-muted-foreground hover:text-foreground transition-colors">
                Browse
              </Link>
              {mounted && user && (
                <>
                  <Link href="/orders" className="text-sm font-medium text-muted-foreground hover:text-foreground transition-colors">
                    My Orders
                  </Link>
                  <Link href="/profile" className="text-sm font-medium text-muted-foreground hover:text-foreground transition-colors">
                    Profile
                  </Link>
                </>
              )}
            </nav>
          </div>

          {/* Actions */}
          <div className="flex items-center space-x-4">
            <ThemeToggle />

            {mounted && user ? (
              <div className="flex items-center space-x-4">
                <span className="text-sm text-muted-foreground hidden sm:inline-block">
                  {user.name || user.email}
                </span>
                <Button onClick={handleLogout} variant="outline" size="sm" className="btn-text">
                  Sign Out
                </Button>
              </div>
            ) : (
              <div className="flex items-center space-x-3">
                <Button asChild variant="ghost" size="sm" className="btn-text">
                  <Link href="/signin">Sign In</Link>
                </Button>
                <Button asChild size="sm" className="btn-text bg-primary hover:bg-primary/90">
                  <Link href="/signup">Sign Up</Link>
                </Button>
              </div>
            )}
          </div>
        </div>
      </div>
    </header>
  );
}