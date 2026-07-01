import type { Metadata } from "next";
import { Inter, Fraunces, IBM_Plex_Mono } from "next/font/google";
import "./globals.css";
import { Providers } from "./providers";

// Inter (body/UI) — variable. Fraunces (display serif) + IBM Plex Mono (numbers) —
// static weights we actually use. All SIL Open Font License, self-hosted by next/font
// (no runtime Google request → zero-cost, ADR-0003). (SPEC-200 FR-2004)
const inter = Inter({
  subsets: ["latin"],
  variable: "--font-inter",
  display: "swap",
});

const fraunces = Fraunces({
  subsets: ["latin"],
  variable: "--font-fraunces",
  display: "swap",
  weight: ["500", "600", "700"],
});

const ibmPlexMono = IBM_Plex_Mono({
  subsets: ["latin"],
  variable: "--font-ibm-plex-mono",
  display: "swap",
  weight: ["500", "600"],
});

export const metadata: Metadata = {
  title: "YieldForge",
  description:
    "Copiloto de investimentos para FIIs e renda fixa — explicável, nunca uma recomendação.",
};

export default function RootLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  return (
    <html
      lang="pt-BR"
      className={`${inter.variable} ${fraunces.variable} ${ibmPlexMono.variable} antialiased`}
    >
      {/* aurora-bg mounts the ambient glow layer (dark-first; dropped on light) */}
      <body className="aurora-bg min-h-screen">
        <Providers>{children}</Providers>
      </body>
    </html>
  );
}
