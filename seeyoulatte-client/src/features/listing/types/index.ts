export interface Listing {
  id: string;
  name: string;
  description?: string;
  createdAt: string;
  updatedAt: string;
}

export interface CreateListingRequest {
  name: string;
  description?: string;
}

export interface UpdateListingRequest {
  name?: string;
  description?: string;
}

export interface ListingListResponse {
  data: Listing[];
  total: number;
  page: number;
  pageSize: number;
}