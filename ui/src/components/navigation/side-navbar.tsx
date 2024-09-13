"use client";
import { Icons } from "@/components/icons";
import { cn } from "@/lib/utils";
import type { NavItemGroup } from "@/types/navigation";
import type { Domain, Permissions } from "@absmach/magistrala-sdk";
import type { Session } from "next-auth";
import Link from "next/link";
import { usePathname } from "next/navigation";
import type { JSX } from "react";
import { DomainSelection } from "./domain-selection";

interface SideNavbarProps {
  domainPermissions: Permissions;
  navGroups: NavItemGroup[];
  isMobile?: boolean;
  domains?: Domain[];
  session?: Session;
  error?: string | null;
  expand: boolean;
  toggleExpand?: () => void;
  expandWidth: string;
  collapseWidth: string;
}

export function SideNavbar({
  domainPermissions,
  navGroups,
  isMobile,
  domains,
  session,
  error,
  expand,
  toggleExpand,
  expandWidth,
  collapseWidth,
}: SideNavbarProps): JSX.Element | null {
  const path = usePathname();
  const SidebarCollapseIcon = Icons.sidebarCollapse;
  const SidebarExpandIcon = Icons.sidebarExpand;
  if (!navGroups?.length) {
    return null;
  }

  return (
    <nav
      className={cn(
        "grid items-start gap-2 h-full ease-in-out transition-all duration-100",
        expand ? expandWidth : collapseWidth,
      )}
    >
      <>
        <div
          className={cn(
            "flex flex-col  relative ease-in-out transition-all duration-100 justify-center h-12 mb-2",
            expand ? expandWidth : collapseWidth,
          )}
        >
          <div className=" self-center py-2 mt-4">
            <DomainSelection
              expand={expand}
              domains={domains}
              session={session}
              isMobile={isMobile}
              error={error}
            />
          </div>
          {!isMobile && (
            <div className="absolute -top-2  right-0">
              <button
                className="p-0 rounded-md text-sm font-medium text-white hover:bg-accent hover:text-accent-foreground"
                onClick={toggleExpand}
                type="button"
              >
                {expand ? (
                  <SidebarCollapseIcon className="h-4 w-4" />
                ) : (
                  <SidebarExpandIcon className="h-4 w-4" />
                )}
              </button>
            </div>
          )}
        </div>
      </>

      {navGroups.map((group) => (
        <div
          key={group.title}
          className={cn(
            "flex flex-col gap-2 transition-all  duration-100 ",
            expand ? `  ${expandWidth}` : ` ${collapseWidth}`,
          )}
        >
          <div className="text-[10px] gap-4 text-center text-white ">
            <hr
              className={cn(
                "pb-2  self-stretch ease-in-out transition-all duration-100 ",
                expand ? expandWidth : collapseWidth,
              )}
            />
            <div
              className={cn(
                "block transition-opacity duration-100 overflow-hidden text-nowrap	",
                expand ? "opacity-100" : "opacity-0",
              )}
            >
              {group.title.toUpperCase()}
            </div>
          </div>
          {group.navItems?.map((item) => {
            const Icon = Icons[item.icon || "arrowRight"];
            const hideItem =
              !domainPermissions.permissions.includes("admin") &&
              item.title === "Invitations";
            return (
              <Link
                className={cn(hideItem && "hidden", "px-2")}
                key={item.href}
                href={item.disabled ? "/" : (item.href as string)}
              >
                <div
                  className={cn(
                    "group rounded-md flex flex-row items-center text-center px-3 py-2 text-sm text-white hover:bg-popover hover:text-popover-foreground midnightsky:hover:bg-accent midnightsky:hover:text-white ease-in-out transition-all duration-100 after:justify-center",
                    expand ? "" : "justify-center",
                    path === item.href
                      ? "bg-accent text-popover-foreground"
                      : "transparent",
                    item.disabled && "cursor-not-allowed opacity-80",
                  )}
                >
                  <Icon className="mr-2 min-h-6 min-w-6 h-6 w-6" />
                  <span
                    className={cn(
                      "overflow-hidden text-nowrap",
                      expand ? "opacity-100 " : "hidden opacity-0",
                    )}
                  >
                    {item.title}
                  </span>
                </div>
              </Link>
            );
          })}
        </div>
      ))}
    </nav>
  );
}
