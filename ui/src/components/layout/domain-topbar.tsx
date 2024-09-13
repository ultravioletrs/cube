"use client";
import { ThemeToggle } from "@/components/layout/theme-toggle";
import { UserNav } from "@/components/layout/user-nav";
import { topbarHeight } from "@/constants/data";
import { cn } from "@/lib/utils";
import { type Session, UserRole } from "@/types/auth";
import type {
  DomainsPage,
  InvitationsPage,
  Permissions,
  User,
} from "@absmach/magistrala-sdk";
import { useEffect, useState } from "react";
import { ViewUserInvitations } from "../invitations/viewinvitations";
import { MobileSidebar } from "./mobile-sidebar";

interface Props {
  domainPermissions: Permissions;
  domainResponse:
    | {
        data: DomainsPage;
        error: null;
      }
    | {
        data: null;
        error: string;
      };
  invitationResponse:
    | {
        data: InvitationsPage;
        error: null;
      }
    | {
        data: null;
        error: string;
      };
  user: User;
  session: Session;
  expand: boolean;
}
export function DomainTopbar({
  domainPermissions,
  domainResponse,
  invitationResponse,
  session,
  expand,
  user,
}: Props) {
  const [isMobile, setIsMobile] = useState(false);
  const mobileBreakPoint = 768;

  const handleResize = () => {
    if (window.innerWidth < mobileBreakPoint) {
      setIsMobile(true);
    } else {
      setIsMobile(false);
    }
  };

  useEffect(() => {
    window.addEventListener("load", () => {
      handleResize();
    });
  });
  const fullWidth = "w-[calc(100vw-12vw)]";
  const collapsedWidth = "w-[calc(100vw-5vw)]";

  return (
    <div
      className={cn(
        "fixed w-full top-0 right-0 supports-backdrop-blur:bg-background/60 border-b bg-card backdrop-blur z-20",
        !isMobile && expand ? fullWidth : collapsedWidth,
      )}
    >
      <nav
        className={`h-${topbarHeight} flex items-center justify-between px-4`}
      >
        <div className="flex items-center justify-start space-x-2">
          <div className={cn("block md:!hidden")}>
            <MobileSidebar
              domainPermissions={domainPermissions}
              domains={(domainResponse?.data as DomainsPage).domains}
              session={session}
              error={domainResponse.error}
            />
          </div>
        </div>

        <div className="flex items-center gap-2">
          <ViewUserInvitations
            invitations={invitationResponse.data as InvitationsPage}
            error={invitationResponse.error}
          />
          <UserNav
            role={
              session.user.role
                ? (session.user.role as UserRole)
                : UserRole.User
            }
          />
          <ThemeToggle user={user} />
        </div>
      </nav>
    </div>
  );
}
