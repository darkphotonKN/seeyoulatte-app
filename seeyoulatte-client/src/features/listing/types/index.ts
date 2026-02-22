export interface Listing {
  id: string;
  seller_id: string;
  title: string;
  name?: string; // For backward compatibility
  description?: string;
  category: 'product' | 'experience';
  price: number;
  quantity: number;
  pickup_instructions?: string;
  expires_at?: string;
  is_active: boolean;
  created_at: string;
  createdAt?: string; // For backward compatibility
  updatedAt?: string;
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