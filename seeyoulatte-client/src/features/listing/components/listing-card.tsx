import { Listing } from "../types";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { MapPin, Package, Calendar } from "lucide-react";
import Link from "next/link";
import Image from "next/image";

interface ListingCardProps {
  listing: Listing;
}

// Helper to get random coffee image placeholder
const getCoffeeImage = (id: string) => {
  // Use listing ID to consistently pick same placeholder
  // Using Unsplash for beautiful coffee images
  const images = [
    'https://images.unsplash.com/photo-1495474472287-4d71bcdd2085?w=800&q=80', // Coffee shop latte
    'https://images.unsplash.com/photo-1514432324607-a09d9b4aefdd?w=800&q=80', // Coffee beans
    'https://images.unsplash.com/photo-1511920170033-f8396924c348?w=800&q=80', // Coffee cup
  ];
  const index = parseInt(id.slice(-1), 16) % 3;
  return images[index];
};

export function ListingCard({ listing }: ListingCardProps) {
  const displayName = listing.title || listing.name || "Untitled";
  const isExperience = listing.category === 'experience';

  // Ensure price is a number
  const price = typeof listing.price === 'number' ? listing.price : parseFloat(listing.price as any) || 0;

  return (
    <Link href={`/listings/${listing.id}`}>
      <Card className="group cursor-pointer overflow-hidden border-0 shadow-sm hover:shadow-xl transition-all duration-300">
        {/* Image Container */}
        <div className="relative aspect-[4/3] overflow-hidden bg-muted">
          <Image
            src={getCoffeeImage(listing.id)}
            alt={displayName}
            fill
            className="object-cover group-hover:scale-110 transition-transform duration-500"
            sizes="(max-width: 768px) 100vw, (max-width: 1200px) 50vw, 33vw"
          />
          {/* Category Badge */}
          <Badge
            variant={isExperience ? "secondary" : "default"}
            className="absolute top-3 left-3 font-sans"
          >
            {isExperience ? "Experience" : "Product"}
          </Badge>

          {/* Quantity Badge */}
          {listing.quantity <= 5 && listing.quantity > 0 && (
            <Badge
              variant="destructive"
              className="absolute top-3 right-3 font-sans"
            >
              Only {listing.quantity} left
            </Badge>
          )}
        </div>

        {/* Content */}
        <div className="p-4 space-y-3">
          {/* Title and Price */}
          <div className="flex justify-between items-start gap-2">
            <h3 className="font-serif text-lg line-clamp-2 group-hover:text-primary transition-colors">
              {displayName}
            </h3>
            <div className="text-right flex-shrink-0">
              <p className="font-sans font-semibold text-lg">
                ${price.toFixed(2)}
              </p>
              {isExperience && (
                <p className="text-xs text-muted-foreground">/person</p>
              )}
            </div>
          </div>

          {/* Description */}
          {listing.description && (
            <p className="text-sm text-muted-foreground line-clamp-2 font-sans">
              {listing.description}
            </p>
          )}

          {/* Meta Information */}
          <div className="flex items-center gap-4 text-xs text-muted-foreground font-sans">
            {listing.pickup_instructions && (
              <div className="flex items-center gap-1">
                <MapPin className="h-3 w-3" />
                <span className="line-clamp-1">Pickup available</span>
              </div>
            )}

            {listing.quantity > 0 && (
              <div className="flex items-center gap-1">
                <Package className="h-3 w-3" />
                <span>{listing.quantity} available</span>
              </div>
            )}
          </div>

          {/* Expiry Warning */}
          {listing.expires_at && (
            <div className="flex items-center gap-1 text-xs text-orange-600 dark:text-orange-400 font-sans">
              <Calendar className="h-3 w-3" />
              <span>
                Expires {new Date(listing.expires_at).toLocaleDateString()}
              </span>
            </div>
          )}
        </div>
      </Card>
    </Link>
  );
}