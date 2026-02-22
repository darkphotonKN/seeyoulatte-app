"use client";

import { useState, useEffect } from "react";
import { useParams, useRouter } from "next/navigation";
import Image from "next/image";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import {
  ArrowLeft,
  ChevronLeft,
  ChevronRight,
  MapPin,
  Package,
  Calendar,
  User,
  Coffee,
} from "lucide-react";
import { listingService } from "@/features/listing/services/api";
import { Listing } from "@/features/listing/types";

// Carousel images using Unsplash placeholders
const carouselImages = [
  'https://images.unsplash.com/photo-1495474472287-4d71bcdd2085?w=1200&q=80',
  'https://images.unsplash.com/photo-1514432324607-a09d9b4aefdd?w=1200&q=80',
  'https://images.unsplash.com/photo-1511920170033-f8396924c348?w=1200&q=80',
  'https://images.unsplash.com/photo-1461023058943-07fcbe16d735?w=1200&q=80',
  'https://images.unsplash.com/photo-1507133750040-4a8f57021571?w=1200&q=80',
];

export default function ListingDetailPage() {
  const params = useParams();
  const router = useRouter();
  const [currentImageIndex, setCurrentImageIndex] = useState(0);
  const [listing, setListing] = useState<Listing | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchListing = async () => {
      try {
        setLoading(true);
        const data = await listingService.getById(params.id as string);
        setListing(data);
      } catch (err) {
        console.error('Failed to fetch listing:', err);
        setError('Failed to load listing details');
      } finally {
        setLoading(false);
      }
    };

    if (params.id) {
      fetchListing();
    }
  }, [params.id]);

  const nextImage = () => {
    setCurrentImageIndex((prev) => (prev + 1) % carouselImages.length);
  };

  const prevImage = () => {
    setCurrentImageIndex(
      (prev) => (prev - 1 + carouselImages.length) % carouselImages.length
    );
  };

  if (loading) {
    return (
      <div className="min-h-screen bg-background">
        <div className="container mx-auto px-4 py-6">
          <Skeleton className="h-8 w-32" />
        </div>
        <div className="container mx-auto px-4 pb-12">
          <div className="grid lg:grid-cols-2 gap-8">
            <div className="space-y-4">
              <Skeleton className="aspect-square rounded-lg" />
              <div className="grid grid-cols-5 gap-2">
                {[...Array(5)].map((_, i) => (
                  <Skeleton key={i} className="aspect-square rounded-md" />
                ))}
              </div>
            </div>
            <div className="space-y-6">
              <Skeleton className="h-8 w-24" />
              <Skeleton className="h-12 w-full" />
              <Skeleton className="h-24 w-full" />
              <Skeleton className="h-32 w-full" />
            </div>
          </div>
        </div>
      </div>
    );
  }

  if (error || !listing) {
    return (
      <div className="min-h-screen bg-background flex items-center justify-center">
        <div className="text-center">
          <h2 className="text-2xl font-semibold mb-2">Oops!</h2>
          <p className="text-muted-foreground">{error || 'Listing not found'}</p>
          <Button
            onClick={() => router.push('/listings')}
            className="mt-4"
          >
            Back to Listings
          </Button>
        </div>
      </div>
    );
  }

  const isExperience = listing.category === "experience";
  const displayName = listing.title || listing.name || "Untitled";
  const price = typeof listing.price === 'number' ? listing.price : parseFloat(listing.price as any) || 0;

  return (
    <div className="min-h-screen bg-background">
      {/* Back Button */}
      <div className="container mx-auto px-4 py-6">
        <Button
          variant="ghost"
          size="sm"
          onClick={() => router.back()}
          className="btn-text"
        >
          <ArrowLeft className="h-4 w-4 mr-2" />
          Back to listings
        </Button>
      </div>

      <div className="container mx-auto px-4 pb-12">
        <div className="grid lg:grid-cols-2 gap-8">
          {/* Image Carousel */}
          <div className="space-y-4">
            <div className="relative aspect-square rounded-lg overflow-hidden bg-muted">
              <Image
                src={carouselImages[currentImageIndex]}
                alt={displayName}
                fill
                className="object-cover"
                priority
              />

              {/* Carousel Controls */}
              <button
                onClick={prevImage}
                className="absolute left-4 top-1/2 -translate-y-1/2 bg-white/80 dark:bg-black/80 rounded-full p-2 hover:bg-white dark:hover:bg-black transition-colors"
              >
                <ChevronLeft className="h-5 w-5" />
              </button>
              <button
                onClick={nextImage}
                className="absolute right-4 top-1/2 -translate-y-1/2 bg-white/80 dark:bg-black/80 rounded-full p-2 hover:bg-white dark:hover:bg-black transition-colors"
              >
                <ChevronRight className="h-5 w-5" />
              </button>

              {/* Image Indicators */}
              <div className="absolute bottom-4 left-1/2 -translate-x-1/2 flex gap-2">
                {carouselImages.map((_, index) => (
                  <button
                    key={index}
                    onClick={() => setCurrentImageIndex(index)}
                    className={`h-2 w-2 rounded-full transition-colors ${
                      index === currentImageIndex
                        ? "bg-white"
                        : "bg-white/50"
                    }`}
                  />
                ))}
              </div>
            </div>

            {/* Thumbnail Strip */}
            <div className="grid grid-cols-5 gap-2">
              {carouselImages.map((img, index) => (
                <button
                  key={index}
                  onClick={() => setCurrentImageIndex(index)}
                  className={`relative aspect-square rounded-md overflow-hidden ${
                    index === currentImageIndex
                      ? "ring-2 ring-primary"
                      : "opacity-70 hover:opacity-100"
                  }`}
                >
                  <Image
                    src={img}
                    alt={`View ${index + 1}`}
                    fill
                    className="object-cover"
                  />
                </button>
              ))}
            </div>
          </div>

          {/* Listing Details */}
          <div className="space-y-6">
            {/* Header */}
            <div>
              <div className="flex items-start justify-between mb-3">
                <Badge
                  variant={isExperience ? "secondary" : "default"}
                  className="font-sans"
                >
                  {isExperience ? "Experience" : "Product"}
                </Badge>
                {listing.quantity <= 5 && listing.quantity > 0 && (
                  <Badge variant="destructive" className="font-sans">
                    Only {listing.quantity} left
                  </Badge>
                )}
              </div>

              <h1 className="heading-secondary mb-4">{displayName}</h1>

              <div className="flex items-baseline gap-4">
                <p className="text-3xl font-semibold font-sans">
                  ${price.toFixed(2)}
                </p>
                {isExperience && (
                  <span className="text-muted-foreground">/person</span>
                )}
              </div>
            </div>

            {/* Description */}
            {listing.description && (
              <div className="space-y-4">
                <h2 className="font-serif text-xl">About this {isExperience ? "experience" : "coffee"}</h2>
                <p className="text-muted-foreground leading-relaxed font-sans">
                  {listing.description}
                </p>
              </div>
            )}

            {/* Details */}
            <Card>
              <CardContent className="pt-6 space-y-4">
                <div className="flex items-start gap-3">
                  <Package className="h-5 w-5 text-muted-foreground mt-0.5" />
                  <div>
                    <p className="font-medium font-sans">Availability</p>
                    <p className="text-sm text-muted-foreground">
                      {listing.quantity} {isExperience ? "slots" : "items"} available
                    </p>
                  </div>
                </div>

                {listing.pickup_instructions && (
                  <div className="flex items-start gap-3">
                    <MapPin className="h-5 w-5 text-muted-foreground mt-0.5" />
                    <div>
                      <p className="font-medium font-sans">Pickup Instructions</p>
                      <p className="text-sm text-muted-foreground">
                        {listing.pickup_instructions}
                      </p>
                    </div>
                  </div>
                )}

                {listing.expires_at && (
                  <div className="flex items-start gap-3">
                    <Calendar className="h-5 w-5 text-muted-foreground mt-0.5" />
                    <div>
                      <p className="font-medium font-sans">Expires</p>
                      <p className="text-sm text-muted-foreground">
                        {new Date(listing.expires_at).toLocaleDateString("en-US", {
                          weekday: "long",
                          year: "numeric",
                          month: "long",
                          day: "numeric",
                        })}
                      </p>
                    </div>
                  </div>
                )}
              </CardContent>
            </Card>

            {/* Seller Info */}
            <Card>
              <CardContent className="pt-6">
                <div className="flex items-start gap-4">
                  <div className="h-12 w-12 rounded-full bg-muted flex items-center justify-center">
                    <User className="h-6 w-6 text-muted-foreground" />
                  </div>
                  <div className="flex-1">
                    <h3 className="font-semibold font-sans">
                      Seller ID: {listing.seller_id.slice(0, 8)}...
                    </h3>
                    <div className="flex items-center gap-2 mt-1">
                      <Coffee className="h-4 w-4 text-primary" />
                      <span className="text-sm">
                        Verified Seller
                      </span>
                    </div>
                    <p className="text-sm text-muted-foreground mt-3">
                      Contact seller for more information about this {isExperience ? "experience" : "product"}.
                    </p>
                  </div>
                </div>
              </CardContent>
            </Card>

            {/* Action Buttons */}
            <div className="flex gap-3">
              <Button size="lg" className="flex-1 btn-text">
                Order Now
              </Button>
              <Button size="lg" variant="outline" className="btn-text">
                Message Seller
              </Button>
            </div>

            {/* Listing Metadata */}
            <div className="text-xs text-muted-foreground">
              <p>Listed on {new Date(listing.created_at || listing.createdAt || '').toLocaleDateString()}</p>
              <p>ID: {listing.id}</p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}