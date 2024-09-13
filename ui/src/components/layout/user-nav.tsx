"use client";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuShortcut,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { cn } from "@/lib/utils";
import { UserRole } from "@/types/auth";
import { signOut, useSession } from "next-auth/react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect } from "react";
export function UserNav({
  role,
}: {
  role: UserRole;
}) {
  const { data: session, status } = useSession();
  const loginUrl = `${process.env.NEXT_PUBLIC_BASE_URL}/login`;
  const route = useRouter();
  useEffect(() => {
    const handleKeydown = (event: KeyboardEvent) => {
      if (event.shiftKey && event.ctrlKey && event.key === "P") {
        event.preventDefault();
        window.location.href = "/profile";
      }
      if (event.shiftKey && event.ctrlKey && event.key === "U") {
        event.preventDefault();
        window.location.href = "/platform-management/users";
      }
      if (event.shiftKey && event.ctrlKey && event.key === "D") {
        event.preventDefault();
        window.location.href = "/domains";
      }
      if (event.shiftKey && event.ctrlKey && event.key === "Q") {
        event.preventDefault();
        signOut({ callbackUrl: loginUrl });
      }
    };

    window.addEventListener("keydown", handleKeydown);
    return () => {
      window.removeEventListener("keydown", handleKeydown);
    };
  }, [loginUrl]);

  if (session && status === "authenticated") {
    return (
      <DropdownMenu>
        <DropdownMenuTrigger asChild={true}>
          <Button
            variant="ghost"
            className="relative h-8 w-8 rounded-full hover:bg-primary/10"
          >
            <Avatar className="h-8 w-8">
              <AvatarImage
                src={session.user?.image ?? ""}
                alt={session.user?.name ?? ""}
              />
              <AvatarFallback className="bg-primary/10 dark:bg-accent">
                {session.user?.name?.[0]}
              </AvatarFallback>
            </Avatar>
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent className="w-56" align="end" forceMount={true}>
          <DropdownMenuLabel className="font-normal">
            <div className="flex flex-col space-y-1">
              <p className="text-sm font-medium leading-none truncate">
                {session.user?.name}
              </p>
              <p className="text-xs leading-none text-muted-foreground">
                {session.user?.email}
              </p>
            </div>
          </DropdownMenuLabel>
          <DropdownMenuSeparator />
          <DropdownMenuGroup>
            <DropdownMenuItem
              asChild={true}
              className="hover:bg-primary/10 dark:hover:bg-accent focus:bg-primary/10"
            >
              <Link href="/profile">
                <span>Profile</span>
                <DropdownMenuShortcut>⇧⌘P</DropdownMenuShortcut>
              </Link>
            </DropdownMenuItem>
            <DropdownMenuItem className="hover:bg-primary/10 dark:hover:bg-accent focus:bg-primary/10">
              Settings
              <DropdownMenuShortcut>⇧⌘S</DropdownMenuShortcut>
            </DropdownMenuItem>
            <DropdownMenuItem
              asChild={true}
              className={cn(
                role !== UserRole.Admin
                  ? "hidden"
                  : "hover:bg-primary/10 dark:hover:bg-accent focus:bg-primary/10",
              )}
            >
              <Link href={"/platform-management/users"}>
                <span>Manage Users</span>
                <DropdownMenuShortcut>⇧⌘U</DropdownMenuShortcut>
              </Link>
            </DropdownMenuItem>
            <DropdownMenuItem
              asChild={true}
              className="hover:bg-primary/10 dark:hover:bg-accent focus:bg-primary/10"
            >
              <Link href={"/domains"}>
                <span>Switch Domains</span>
                <DropdownMenuShortcut>⇧⌘D</DropdownMenuShortcut>
              </Link>
            </DropdownMenuItem>
            <DropdownMenuItem
              asChild={true}
              className={cn(
                role !== UserRole.Admin
                  ? "hidden"
                  : "hover:bg-primary/10 dark:hover:bg-accent focus:bg-primary/10",
              )}
            >
              <Link href={"/platform-management"}>
                <span>Platform Management</span>
                <DropdownMenuShortcut>⇧⌘M</DropdownMenuShortcut>
              </Link>
            </DropdownMenuItem>
          </DropdownMenuGroup>
          <DropdownMenuSeparator />
          <DropdownMenuItem
            onClick={async () => {
              const data = await signOut({
                redirect: false,
                callbackUrl: loginUrl,
              });
              route.push(data.url);
            }}
            className="hover:bg-primary/10 dark:hover:bg-accent focus:bg-primary/10"
          >
            Log out
            <DropdownMenuShortcut>⇧⌘Q</DropdownMenuShortcut>
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    );
  }
}
