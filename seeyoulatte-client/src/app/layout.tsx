import type { Metadata } from "next";
import { Inter, Crimson_Text } from "next/font/google";
import "./globals.css";
import { Providers } from "@/components/providers";
import { Toaster } from "@/components/ui/toaster";
import { ThemeProvider } from "@/components/theme-provider";

const inter = Inter({
  subsets: ["latin"],
  variable: "--font-sans",
  display: 'swap',
});

const crimsonText = Crimson_Text({
  weight: ['400', '600', '700'],
  subsets: ["latin"],
  variable: "--font-serif",
  display: 'swap',
});

export const metadata: Metadata = {
  title: "SeeYouLatte - Share Your Coffee",
  description: "A peer-to-peer coffee marketplace. Share your home roasts and coffee experiences.",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body className={`${inter.variable} ${crimsonText.variable} font-sans antialiased`}>
        <ThemeProvider>
          <Providers>
            {children}
            <Toaster />
          </Providers>
        </ThemeProvider>
      </body>
    </html>
  );
}