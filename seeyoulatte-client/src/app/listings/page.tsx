"use client";

import { useState } from "react";
import { useListingList } from "@/features/listing/hooks/use-listing";
import { ListingList } from "@/features/listing/components/listing-list";
import { ListingCreateDialog } from "@/features/listing/components/listing-create-dialog";
import { Button } from "@/components/ui/button";
import { PlusCircle } from "lucide-react";

export default function ListingsPage() {
  const [page, setPage] = useState(1);
  const [createOpen, setCreateOpen] = useState(false);
  const { data, isLoading, error } = useListingList(page, 10);

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
    <div className="container mx-auto py-10">
      <div className="flex justify-between listings-center mb-6">
        <h1 className="text-3xl font-bold">Listings</h1>
        <Button onClick={() => setCreateOpen(true)}>
          <PlusCircle className="mr-2 h-4 w-4" />
          Add Listing
        </Button>
      </div>

      <ListingList
        listings={data?.data || []}
        isLoading={isLoading}
        page={page}
        totalPages={Math.ceil((data?.total || 0) / 10)}
        onPageChange={setPage}
      />

      <ListingCreateDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
      />
    </div>
  );
}