"use client";

import { useState } from "react";
import { useListingList } from "@/features/listing/hooks/use-listing";
import { ListingList } from "@/features/listing/components/listing-list";
import { ListingCreateDialog } from "@/features/listing/components/listing-create-dialog";
import { Button } from "@/components/ui/button";
import { PlusCircle } from "lucide-react";
import { Header } from "@/components/header";

export default function ListingsPage() {
  const [page, setPage] = useState(1);
  const [createOpen, setCreateOpen] = useState(false);
  const { data, isLoading, error } = useListingList(page, 20); // Increased page size for grid

  if (error) {
    return (
      <div className="container mx-auto py-10">
        <div className="text-center text-red-500">
          Error loading listings: {error.message}
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-background">
      <Header />
      {/* Hero Section */}
      <div className="border-b">
        <div className="container mx-auto px-4 py-12">
          <div className="max-w-3xl">
            <h1 className="heading-primary mb-4">Discover Local Coffee</h1>
            <p className="text-lg text-muted-foreground font-sans">
              From freshly roasted beans to unique brewing experiences, find your perfect coffee match in your neighborhood.
            </p>
          </div>
        </div>
      </div>

      {/* Main Content */}
      <div className="container mx-auto px-4 py-8">
        <div className="flex justify-between items-center mb-8">
          <div className="flex items-center gap-4">
            <span className="text-sm text-muted-foreground font-sans">
              {data?.total || 0} listings available
            </span>
          </div>
          <Button
            onClick={() => setCreateOpen(true)}
            className="btn-text rounded-full"
            size="lg"
          >
            <PlusCircle className="mr-2 h-4 w-4" />
            Share Your Coffee
          </Button>
        </div>

        <ListingList
          listings={data?.data || []}
          isLoading={isLoading}
          page={page}
          totalPages={Math.ceil((data?.total || 0) / 20)}
          onPageChange={setPage}
        />

        <ListingCreateDialog
          open={createOpen}
          onOpenChange={setCreateOpen}
        />
      </div>
    </div>
  );
}