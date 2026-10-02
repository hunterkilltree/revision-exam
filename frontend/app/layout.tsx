import type { Metadata, Viewport } from "next";
import "@/styles/tokens.css";

export const metadata: Metadata = { title: "Live Quiz Sessions" };
export const viewport: Viewport = { width: "device-width", initialScale: 1 };

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <head>
        {/* eslint-disable-next-line @next/next/no-page-custom-font */}
        <link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=Bricolage+Grotesque:wght@600;700&family=IBM+Plex+Mono:wght@500;700&family=IBM+Plex+Sans:wght@400;600&display=swap" />
      </head>
      <body>{children}</body>
    </html>
  );
}
