import { apiClient } from '@/lib/api/client';

export interface CreatePaymentRequest {
  order_id: string;
  payment_method: 'card' | 'bank_transfer';
  token?: string;
  return_url?: string;
}

export interface CreatePaymentResponse {
  payment_id: string;
  status: string;
  amount: number;
  currency: string;
  client_secret?: string;
  redirect_url?: string;
  next_action?: string;
}

export const paymentService = {
  createPayment: async (data: CreatePaymentRequest): Promise<CreatePaymentResponse> => {
    const response = await apiClient.post<CreatePaymentResponse>('/api/payments', data);
    return response.data;
  },
};