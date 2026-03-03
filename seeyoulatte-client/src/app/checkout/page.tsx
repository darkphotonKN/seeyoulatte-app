'use client';

import { useSearchParams, useRouter } from 'next/navigation';
import { Suspense, useEffect, useState } from 'react';
import CheckoutForm from '@/components/checkout/CheckoutForm';
import { CheckoutData } from '@/features/payment/types';
import { Card, CardContent } from '@/components/ui/card';
import { Loader2 } from 'lucide-react';

function CheckoutContent() {
  const searchParams = useSearchParams();
  const router = useRouter();
  const [checkoutData, setCheckoutData] = useState<CheckoutData | null>(null);
  const [orderId, setOrderId] = useState<string>('');

  useEffect(() => {
    // Parse checkout data from URL params
    const listing_id = searchParams.get('listing_id');
    const listing_title = searchParams.get('title');
    const quantity = searchParams.get('quantity');
    const price = searchParams.get('price');
    const seller_name = searchParams.get('seller');
    const order_id = searchParams.get('order_id');

    if (!listing_id || !listing_title || !quantity || !price || !seller_name) {
      // Redirect back if missing data
      router.push('/listings');
      return;
    }

    const quantityNum = parseInt(quantity);
    const priceNum = parseFloat(price);

    setCheckoutData({
      listing_id,
      listing_title,
      quantity: quantityNum,
      price: priceNum,
      total: quantityNum * priceNum,
      seller_name,
    });

    // For now, we'll use a placeholder order ID
    // In a real implementation, you'd create the order first
    setOrderId(order_id || `order_${Date.now()}`);
  }, [searchParams, router]);

  const handlePaymentSuccess = () => {
    // Redirect to success page or order details
    router.push('/orders?payment=success');
  };

  if (!checkoutData) {
    return (
      <div className="flex items-center justify-center min-h-[60vh]">
        <Card className="w-full max-w-md">
          <CardContent className="pt-6">
            <div className="flex items-center justify-center space-x-2">
              <Loader2 className="h-6 w-6 animate-spin" />
              <span>Loading checkout...</span>
            </div>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="container max-w-2xl mx-auto py-8">
      <h1 className="text-3xl font-bold mb-8">Checkout</h1>
      <CheckoutForm
        checkoutData={checkoutData}
        orderId={orderId}
        onSuccess={handlePaymentSuccess}
      />
    </div>
  );
}

export default function CheckoutPage() {
  return (
    <Suspense
      fallback={
        <div className="flex items-center justify-center min-h-[60vh]">
          <Card className="w-full max-w-md">
            <CardContent className="pt-6">
              <div className="flex items-center justify-center space-x-2">
                <Loader2 className="h-6 w-6 animate-spin" />
                <span>Loading...</span>
              </div>
            </CardContent>
          </Card>
        </div>
      }
    >
      <CheckoutContent />
    </Suspense>
  );
}