import Breadcrumbs, { type BreadcrumbProps } from "@/components/breadcrumbs";
import { SidebarNav } from "@/components/navigation/tab-navbar";
import { getServerSession } from "@/lib/nextauth";
import type { NavItem } from "@/types/navigation";
import type { ReactNode } from "react";

export default async function PlatformManagementLayout({
  children,
}: {
  children: ReactNode;
}) {
  const sidebarNavItems: NavItem[] = [
    {
      title: "Domains",
      href: "/platform-management",
      icon: "domains",
    },
    {
      title: "Users",
      href: "/platform-management/users",
      icon: "users",
    },
  ];
  const session = await getServerSession();
  let breadcrumb: BreadcrumbProps[] = [];
  if (session.domain) {
    breadcrumb = [
      { label: "HomePage", href: "/domain/info" },
      {
        label: "Platform Management",
        href: "/platform-management",
        active: true,
      },
    ];
  } else {
    breadcrumb = [
      { label: "Domain Login", href: "/" },
      {
        label: "Platform Management",
        href: "/platform-management",
        active: true,
      },
    ];
  }

  return (
    <div className="flex flex-row w-[100%]">
      <div className="w-full pt-14 pb-2 mt-4">
        <div className="px-10">
          <Breadcrumbs breadcrumbs={breadcrumb} />
          <div className="flex flex-col space-y-8 lg:flex-row lg:space-y-0 py-4 md:py-6">
            <aside className="lg:w-1/6">
              <SidebarNav items={sidebarNavItems} />
            </aside>
            <div className="flex-1 md:container w-full mx-auto mt-4 pb-4 md:pb-10">
              {children}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
