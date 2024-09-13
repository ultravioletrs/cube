import type { ReactNode } from "react";

export default function AuthRootLayout({ children }: { children: ReactNode }) {
  return (
    <>
      <div className="flex h-screen overflow-hidden">
        <main className="w-full pt-24 px-4 md:px-0">{children}</main>
      </div>
    </>
  );
}
