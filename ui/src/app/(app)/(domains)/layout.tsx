import HomeTopbar from "@/components/layout/home-topbar";
import type { ReactNode } from "react";
export default function Layout({ children }: { children: ReactNode }) {
  return (
    <>
      <HomeTopbar />
      {children}
    </>
  );
}
