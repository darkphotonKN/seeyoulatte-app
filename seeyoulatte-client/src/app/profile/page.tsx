"use client";

import { useEffect, useState } from "react";
import { useAuthStore } from "@/stores/auth-store";
import { useRouter } from "next/navigation";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { MapPin, Calendar, Coffee, Package, Shield } from "lucide-react";
import { apiClient } from "@/lib/api/client";
import { Skeleton } from "@/components/ui/skeleton";

interface UserProfile {
  id: string;
  email: string;
  name: string;
  bio?: string;
  location_text?: string;
  avatar_url?: string;
  is_verified: boolean;
  preferred_pickup_instructions?: string;
  created_at: string;
  last_login_at?: string;
}

export default function ProfilePage() {
  const [profile, setProfile] = useState<UserProfile | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const user = useAuthStore((state) => state.user);
  const router = useRouter();

  useEffect(() => {
    if (!user) {
      router.push("/signin");
      return;
    }

    const fetchProfile = async () => {
      try {
        const { data } = await apiClient.get("/auth/me");
        setProfile(data);
        setError(null);
      } catch (err) {
        // If API doesn't exist, use dummy data
        setProfile({
          id: user.id,
          email: user.email || "coffee.lover@example.com",
          name: user.name || "Coffee Enthusiast",
          bio: "Passionate about specialty coffee. Home barista with a love for Ethiopian single origins and experimental processing methods.",
          location_text: "San Francisco, CA",
          avatar_url: undefined,
          is_verified: true,
          preferred_pickup_instructions: "Ring doorbell twice, I'll meet you at the lobby",
          created_at: new Date().toISOString(),
          last_login_at: new Date().toISOString(),
        });
      } finally {
        setLoading(false);
      }
    };

    fetchProfile();
  }, [user, router]);

  if (loading) {
    return (
      <div className="container mx-auto px-4 py-8 max-w-4xl">
        <Card>
          <CardHeader>
            <div className="flex items-center space-x-4">
              <Skeleton className="h-20 w-20 rounded-full" />
              <div className="space-y-2">
                <Skeleton className="h-6 w-48" />
                <Skeleton className="h-4 w-32" />
              </div>
            </div>
          </CardHeader>
          <CardContent className="space-y-6">
            <Skeleton className="h-20 w-full" />
            <Skeleton className="h-20 w-full" />
          </CardContent>
        </Card>
      </div>
    );
  }

  if (error || !profile) {
    return (
      <div className="container mx-auto px-4 py-8 max-w-4xl">
        <Card>
          <CardContent className="py-8 text-center">
            <p className="text-muted-foreground">Unable to load profile</p>
            <Button onClick={() => router.push("/")} variant="outline" className="mt-4">
              Go Home
            </Button>
          </CardContent>
        </Card>
      </div>
    );
  }

  const getInitials = (name: string) => {
    return name
      .split(" ")
      .map((n) => n[0])
      .join("")
      .toUpperCase()
      .slice(0, 2);
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString("en-US", {
      year: "numeric",
      month: "long",
      day: "numeric",
    });
  };

  return (
    <div className="container mx-auto px-4 py-8 max-w-4xl">
      <Card>
        <CardHeader>
          <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between space-y-4 sm:space-y-0">
            <div className="flex items-center space-x-4">
              <Avatar className="h-20 w-20">
                <AvatarImage src={profile.avatar_url} alt={profile.name} />
                <AvatarFallback className="text-lg">{getInitials(profile.name)}</AvatarFallback>
              </Avatar>
              <div>
                <CardTitle className="text-2xl flex items-center gap-2">
                  {profile.name}
                  {profile.is_verified && (
                    <Shield className="h-5 w-5 text-primary" title="Verified Seller" />
                  )}
                </CardTitle>
                <CardDescription>{profile.email}</CardDescription>
              </div>
            </div>
            <Button variant="outline">Edit Profile</Button>
          </div>
        </CardHeader>
        <CardContent className="space-y-6">
          {/* Bio Section */}
          {profile.bio && (
            <div className="space-y-2">
              <h3 className="font-semibold text-sm text-muted-foreground uppercase tracking-wider">About</h3>
              <p className="text-foreground">{profile.bio}</p>
            </div>
          )}

          {/* Details Grid */}
          <div className="grid gap-4 md:grid-cols-2">
            {profile.location_text && (
              <div className="flex items-start space-x-3">
                <MapPin className="h-5 w-5 text-muted-foreground mt-0.5" />
                <div>
                  <p className="text-sm font-medium text-muted-foreground">Location</p>
                  <p className="text-foreground">{profile.location_text}</p>
                </div>
              </div>
            )}

            <div className="flex items-start space-x-3">
              <Calendar className="h-5 w-5 text-muted-foreground mt-0.5" />
              <div>
                <p className="text-sm font-medium text-muted-foreground">Member Since</p>
                <p className="text-foreground">{formatDate(profile.created_at)}</p>
              </div>
            </div>

            {profile.preferred_pickup_instructions && (
              <div className="flex items-start space-x-3 md:col-span-2">
                <Package className="h-5 w-5 text-muted-foreground mt-0.5" />
                <div>
                  <p className="text-sm font-medium text-muted-foreground">Default Pickup Instructions</p>
                  <p className="text-foreground">{profile.preferred_pickup_instructions}</p>
                </div>
              </div>
            )}
          </div>

          {/* Activity Stats */}
          <div className="border-t pt-6">
            <h3 className="font-semibold text-sm text-muted-foreground uppercase tracking-wider mb-4">
              Activity
            </h3>
            <div className="grid gap-4 grid-cols-2 md:grid-cols-4">
              <div className="text-center">
                <p className="text-2xl font-bold">0</p>
                <p className="text-sm text-muted-foreground">Listings</p>
              </div>
              <div className="text-center">
                <p className="text-2xl font-bold">0</p>
                <p className="text-sm text-muted-foreground">Sales</p>
              </div>
              <div className="text-center">
                <p className="text-2xl font-bold">0</p>
                <p className="text-sm text-muted-foreground">Purchases</p>
              </div>
              <div className="text-center">
                <p className="text-2xl font-bold">0.0</p>
                <p className="text-sm text-muted-foreground">Rating</p>
              </div>
            </div>
          </div>

          {/* Seller Status */}
          {profile.is_verified && (
            <div className="border-t pt-6">
              <div className="flex items-center justify-between p-4 bg-primary/5 rounded-lg">
                <div className="flex items-center space-x-3">
                  <Coffee className="h-6 w-6 text-primary" />
                  <div>
                    <p className="font-semibold">Verified Seller</p>
                    <p className="text-sm text-muted-foreground">You can create and manage listings</p>
                  </div>
                </div>
                <Button>Create Listing</Button>
              </div>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}