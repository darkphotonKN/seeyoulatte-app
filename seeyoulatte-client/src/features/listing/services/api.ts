import { apiClient } from "@/lib/api/client";
import { API_ENDPOINTS } from "@/lib/api/endpoints";
import type {
  Listing,
  CreateListingRequest,
  UpdateListingRequest,
  ListingListResponse,
} from "../types";

export const listingService = {
  // Get all listings
  getAll: async (page = 1, pageSize = 10): Promise<ListingListResponse> => {
    const { data } = await apiClient.get(API_ENDPOINTS.ITEM.LIST, {
      params: { page, pageSize },
    });
    return data;
  },

  // Get a single listing
  getById: async (id: string): Promise<Listing> => {
    const { data } = await apiClient.get(API_ENDPOINTS.ITEM.GET(id));
    return data;
  },

  // Create a new listing
  create: async (payload: CreateListingRequest): Promise<Listing> => {
    const { data } = await apiClient.post(API_ENDPOINTS.ITEM.CREATE, payload);
    return data;
  },

  // Update an existing listing
  update: async (id: string, payload: UpdateListingRequest): Promise<Listing> => {
    const { data } = await apiClient.put(API_ENDPOINTS.ITEM.UPDATE(id), payload);
    return data;
  },

  // Delete an listing
  delete: async (id: string): Promise<void> => {
    await apiClient.delete(API_ENDPOINTS.ITEM.DELETE(id));
  },
};