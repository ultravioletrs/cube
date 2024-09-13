"use client";

import { SideNavbar } from "@/components/navigation/side-navbar";
import { Sheet, SheetContent, SheetTrigger } from "@/components/ui/sheet";
import { sideNavBarGroups } from "@/constants/data";
import type { Domain, Permissions } from "@absmach/magistrala-sdk";
import { Separator } from "@radix-ui/react-dropdown-menu";
import { MenuIcon } from "lucide-react";
import type { Session } from "next-auth";
import { getImageProps } from "next/image";
import Link from "next/link";
import { useState } from "react";

export const MobileSidebar = ({
  domainPermissions,
  error,
  domains,
  session,
}: {
  domainPermissions: Permissions;
  error: string | null;
  domains: Domain[];
  session: Session;
}) => {
  const [open, setOpen] = useState(false);
  const expandWidth = "100vw";
  const collapseWidth = "100vw";
  const common = { alt: "Abstract Machines Logo", sizes: "100vw" };
  const {
    props: { srcSet: mobile },
  } = getImageProps({
    ...common,
    width: 64,
    height: 64,
    quality: 70,
    src: "/abstract-machines_logo_square-white.svg",
    className: "dark:invert",
  });

  const {
    props: { srcSet: desktop, ...rest },
  } = getImageProps({
    ...common,
    width: 180,
    height: 64,
    quality: 70,
    src: "/abstract-machines_logo_landscape-white.svg",
    className: "dark:invert",
  });
  return (
    <>
      <Sheet open={open} onOpenChange={setOpen}>
        <SheetTrigger asChild={true}>
          <MenuIcon />
        </SheetTrigger>
        <SheetContent side="left" className="!px-0 w-94 bg-sidebar">
          <div className="">
            <div className="min-h-14">
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
            </div>
            <Separator />
          </div>
          <div className=" py-1">
            <SideNavbar
              domainPermissions={domainPermissions}
              navGroups={sideNavBarGroups}
              isMobile={true}
              domains={domains}
              session={session}
              collapseWidth={collapseWidth}
              expandWidth={expandWidth}
              expand={true}
              error={error}
            />
          </div>
        </SheetContent>
      </Sheet>
    </>
  );
};
