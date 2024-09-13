import SessionMaintainer from "@/components/layout/session-maintainer";
import SessionProvider from "@/components/providers/next-auth-provider";
import { getServerSession } from "@/lib/nextauth";
import type { ReactNode } from "react";

export default async function AppRootLayout({
  children,
}: {
  children: ReactNode;
}) {
  const session = await getServerSession();

  return (
    <SessionProvider session={session}>
      <SessionMaintainer>{children}</SessionMaintainer>
    </SessionProvider>
  );
}
