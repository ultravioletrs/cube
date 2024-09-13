"use client";

import { DomainTopbar } from "@/components/layout/domain-topbar";
import { sideNavBarGroups } from "@/constants/data";
import type { Session } from "@/types/auth";
import type {
  DomainsPage,
  InvitationsPage,
  Permissions,
  User,
} from "@absmach/magistrala-sdk";
import { getImageProps } from "next/image";
import Link from "next/link";
import { useState } from "react";
import { SideNavbar } from "../navigation/side-navbar";
import { ScrollArea } from "../ui/scroll-area";
import { Separator } from "../ui/separator";

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
  session: Session;
  user: User;
}

export const Navbars = ({
  domainPermissions,
  domainResponse,
  invitationResponse,
  session,
  user,
}: Props) => {
  const [expand, setExpand] = useState<boolean>(true);
  const toggleExpand = () => {
    setExpand((prevExpand) => !prevExpand);
  };
  const expandWidth = "w-[12vw]";
  const collapseWidth = "w-[5vw]";

  const common = { alt: "Abstract Machines Logo", sizes: "100vw" };
  const {
    props: { srcSet: mobile },
  } = getImageProps({
    ...common,
    width: 64,
    height: 64,
    quality: 70,
    src: "/abstract-machines_logo_square-white.svg",
  });

  const {
    props: { srcSet: desktop, ...rest },
  } = getImageProps({
    ...common,
    width: 180,
    height: 64,
    quality: 70,
    src: "/abstract-machines_logo_landscape-white.svg",
  });

  return (
    <>
      <DomainTopbar
        domainPermissions={domainPermissions}
        invitationResponse={invitationResponse}
        domainResponse={domainResponse}
        session={session}
        expand={expand}
        user={user}
      />
      <aside className="fixed bg-sidebar md:top-0 z-50 hidden h-[calc(100vh)] shrink-0 md:sticky md:block">
        <ScrollArea>
          <nav className="hidden border-r h-[100vh] md:block transition ease-in-out delay-1000 z-50">
            <div className="min-h-14">
              {expand && (
                <Link
                  href={"https://github.com/absmach/magistrala"}
                  target="_blank"
                >
                  <picture>
                    <source media={collapseWidth} srcSet={mobile} />
                    <source media={expandWidth} srcSet={desktop} />
                    <img {...rest} alt="Abstract Machines Logo" />
                  </picture>
                </Link>
              )}
            </div>
            <Separator />
            <div className="py-2">
              <SideNavbar
                domainPermissions={domainPermissions}
                navGroups={sideNavBarGroups}
                domains={(domainResponse.data as DomainsPage).domains}
                session={session}
                error={domainResponse.error}
                expand={expand}
                toggleExpand={toggleExpand}
                expandWidth={expandWidth}
                collapseWidth={collapseWidth}
              />
            </div>
          </nav>
        </ScrollArea>
      </aside>
    </>
  );
};
