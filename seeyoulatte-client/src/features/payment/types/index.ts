export interface Payment {
  id: string;
  order_id: string;
  user_id: string;
  amount: number;
  currency: string;
  status: 'pending' | 'processing' | 'succeeded' | 'failed' | 'canceled' | 'refunded';
  payment_method: string;
  stripe_payment_id?: string;
  stripe_charge_id?: string;
  failure_reason?: string;
  processed_at?: string;
  created_at: string;
  updated_at: string;
}

export interface CheckoutData {
  listing_id: string;
  listing_title: string;
  quantity: number;
  price: number;
  total: number;
  seller_name: string;
}