import "@/app/globals.css";
import ThemeProvider from "@/components/providers/theme-provider";
import { metadataVariables } from "@/constants/metadata-variables";
import { ThemeArray, Themes } from "@/types/entities";
import type { Metadata } from "next";
import { Inter } from "next/font/google";
import type { ReactNode } from "react";
import { Toaster } from "sonner";

const inter = Inter({ subsets: ["latin"] });
const baseUrl = `${metadataVariables.baseUrl}/login`;
const title = `${metadataVariables.magistrala}`;
const description =
  "Magistrala UI bridges users and IoT devices, providing seamless connectivity over HTTP, MQTT, WebSocket, and CoAP protocols.Ideal for building complex IoT solutions with reliable, scalable, and efficient middleware.";

export const metadata: Metadata = {
  metadataBase: new URL(baseUrl),
  title: {
    default: title,
    template: `${title} | %s`,
  },
  description: description,
  keywords: "IoT devices, IoT Device Management, IoT Device Connectivity",
  icons: {
    icon: [
      {
        media: "(prefers-color-scheme: light)",
        url: "/abstract-machines_logo_square-light.svg",
        href: "/abstract-machines_logo_square-ligt.svg",
      },
      {
        media: "(prefers-color-scheme: dark)",
        url: "/abstract-machines_logo_square-black.svg",
        href: "/abstract-machines_logo_square-black.svg",
      },
    ],
  },
  openGraph: {
    title: {
      default: title,
      template: `${title} | %s`,
    },
    description: description,
    url: baseUrl,
    type: "website",
    images: metadataVariables.image,
  },
};

export default function RootLayout({ children }: { children: ReactNode }) {
  return (
    <html lang="en" suppressHydrationWarning={true}>
      <link
        rel="icon"
        href="/abstract-machines_logo_square-black.svg"
        sizes="any"
      />
      <body
        className={`${inter.className} dark:bg-background tealtide:bg-background/10 text-popover-foreground`}
      >
        <ThemeProvider
          attribute="class"
          enableColorScheme={true}
          defaultTheme={Themes.Default}
          themes={ThemeArray}
        >
          {children}
        </ThemeProvider>
        <Toaster
          richColors={true}
          expand={true}
          visibleToasts={1}
          closeButton={true}
        />
      </body>
    </html>
  );
}
