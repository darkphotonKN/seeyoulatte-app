import Link from "next/link";
import { Button } from "@/components/ui/button";
import { Coffee, Users, ShoppingBag, ArrowRight } from "lucide-react";

export default function HomePage() {
  return (
    <div className="min-h-screen bg-background">
      {/* Hero Section */}
      <div className="relative">
        <div className="absolute inset-0 bg-gradient-to-br from-accent/20 via-background to-background" />
        <div className="relative container mx-auto px-4 py-24 lg:py-32">
          <div className="flex flex-col items-center text-center space-y-8 max-w-4xl mx-auto">
            <h1 className="heading-primary lg:text-7xl text-foreground">
              Share your coffee.
              <span className="block mt-2">Setup your own shop.</span>
            </h1>

            <p className="text-body text-lg lg:text-xl text-muted-foreground max-w-2xl">
              Join a community of coffee lovers. Sell your home roasts, share your brewing expertise,
              or discover unique beans from passionate local roasters.
            </p>

            <div className="flex flex-col sm:flex-row gap-4 mt-8">
              <Button size="lg" className="btn-text text-lg px-8 py-6" asChild>
                <Link href="/signup">
                  Create Account Now
                </Link>
              </Button>
              <Button size="lg" variant="outline" className="btn-text text-lg px-8 py-6" asChild>
                <Link href="/signin">
                  Login
                </Link>
              </Button>
            </div>
          </div>
        </div>
      </div>

      {/* Features Section */}
      <div className="container mx-auto px-4 py-24">
        <div className="grid md:grid-cols-3 gap-12 max-w-5xl mx-auto">
          <div className="flex flex-col items-center text-center space-y-4">
            <div className="p-4 rounded-full bg-card">
              <Coffee className="h-8 w-8 text-primary" />
            </div>
            <h3 className="heading-tertiary">Sell Your Roasts</h3>
            <p className="text-body text-muted-foreground">
              Share your carefully crafted beans with coffee enthusiasts in your area.
            </p>
          </div>

          <div className="flex flex-col items-center text-center space-y-4">
            <div className="p-4 rounded-full bg-card">
              <Users className="h-8 w-8 text-primary" />
            </div>
            <h3 className="heading-tertiary">Coffee Experiences</h3>
            <p className="text-body text-muted-foreground">
              Invite others to experience your brewing setup and share your coffee knowledge.
            </p>
          </div>

          <div className="flex flex-col items-center text-center space-y-4">
            <div className="p-4 rounded-full bg-card">
              <ShoppingBag className="h-8 w-8 text-primary" />
            </div>
            <h3 className="heading-tertiary">Local Pickup</h3>
            <p className="text-body text-muted-foreground">
              Simple, safe transactions with pickup instructions for every order.
            </p>
          </div>
        </div>
      </div>

      {/* CTA Section */}
      <div className="border-t border-border">
        <div className="container mx-auto px-4 py-16">
          <div className="flex flex-col items-center space-y-6 text-center">
            <h2 className="heading-secondary">Ready to start your coffee journey?</h2>
            <Button size="lg" className="btn-text group" asChild>
              <Link href="/listings">
                Browse Listings
                <ArrowRight className="ml-2 h-4 w-4 transition-transform group-hover:translate-x-1" />
              </Link>
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}