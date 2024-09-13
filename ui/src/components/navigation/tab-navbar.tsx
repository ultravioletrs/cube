"use client";

import { Icons } from "@/components/icons";
import { buttonVariants } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import type { NavItem } from "@/types/navigation";
import Link from "next/link";
import { usePathname } from "next/navigation";
import type { HTMLAttributes } from "react";

interface SidebarNavProps extends HTMLAttributes<HTMLElement> {
  items: NavItem[];
}

export function SidebarNav({ className, items, ...props }: SidebarNavProps) {
  const pathname = usePathname();
  return (
    <nav
      className={cn(
        "flex space-x-2 lg:flex-col lg:space-x-0 lg:space-y-1",
        className,
      )}
      {...props}
    >
      {items.map((item) => {
        const Icon = Icons[item.icon as keyof typeof Icons];
        return (
          <Link
            key={item.href}
            href={item.href as string}
            className={cn(
              buttonVariants({ variant: "ghost" }),
              pathname === item.href
                ? "bg-primary text-white hover:bg-primary/90 midnightsky:hover-bg-accent hover:text-white"
                : "hover:bg-primary/90 midnightsky:hover-accent/10",
              "justify-start rounded-md hover:text-white",
            )}
          >
            <Icon className=" mr-2 min-h-6 min-w-6 h-6 w-6" />
            <span className="hidden sm:block">{item.title}</span>
          </Link>
        );
      })}
    </nav>
  );
}
