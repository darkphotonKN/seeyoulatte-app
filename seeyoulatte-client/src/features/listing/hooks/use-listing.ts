import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { listingService } from "../services/api";
import type { CreateListingRequest, UpdateListingRequest } from "../types";
import { useToast } from "@/components/ui/use-toast";

const QUERY_KEY = "listings";

// Hook to fetch all listings
export const useListingList = (page = 1, pageSize = 10) => {
  return useQuery({
    queryKey: [QUERY_KEY, "list", page, pageSize],
    queryFn: () => listingService.getAll(page, pageSize),
  });
};

// Hook to fetch a single listing
export const useListing = (id: string) => {
  return useQuery({
    queryKey: [QUERY_KEY, id],
    queryFn: () => listingService.getById(id),
    enabled: !!id,
  });
};

// Hook to create an listing
export const useCreateListing = () => {
  const queryClient = useQueryClient();
  const { toast } = useToast();

  return useMutation({
    mutationFn: (data: CreateListingRequest) => listingService.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [QUERY_KEY] });
      toast({
        title: "Success",
        description: "Listing created successfully",
      });
    },
    onError: (error: Error) => {
      toast({
        title: "Error",
        description: error.message || "Failed to create listing",
        variant: "destructive",
      });
    },
  });
};

// Hook to update an listing
export const useUpdateListing = () => {
  const queryClient = useQueryClient();
  const { toast } = useToast();

  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateListingRequest }) =>
      listingService.update(id, data),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: [QUERY_KEY] });
      queryClient.invalidateQueries({ queryKey: [QUERY_KEY, id] });
      toast({
        title: "Success",
        description: "Listing updated successfully",
      });
    },
    onError: (error: Error) => {
      toast({
        title: "Error",
        description: error.message || "Failed to update listing",
        variant: "destructive",
      });
    },
  });
};

// Hook to delete an listing
export const useDeleteListing = () => {
  const queryClient = useQueryClient();
  const { toast } = useToast();

  return useMutation({
    mutationFn: (id: string) => listingService.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [QUERY_KEY] });
      toast({
        title: "Success",
        description: "Listing deleted successfully",
      });
    },
    onError: (error: Error) => {
      toast({
        title: "Error",
        description: error.message || "Failed to delete listing",
        variant: "destructive",
      });
    },
  });
};