import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { cn } from "@/lib/utils";
import { ArrowRightLeft, TriangleAlert } from "lucide-react";

import { Button, buttonVariants } from "@/components/ui/button";
import { DomainLogin } from "@/lib/actions";
import type { Domain } from "@absmach/magistrala-sdk";
import type { Session } from "next-auth";
import Link from "next/link";
import { useState } from "react";
import { Separator } from "../ui/separator";

export function DomainSelection({
  expand,
  domains,
  session,
  isMobile,
  error,
}: {
  expand: boolean;
  domains?: Domain[];
  session?: Session;
  isMobile?: boolean;
  error?: string | null;
}) {
  const [open, setOpen] = useState(false);
  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild={true}>
        <div
          className={cn(
            "flex flex-row justify-between items-center",
            expand ? "gap-2" : "gap-0",
          )}
        >
          {session?.domain ? (
            <div>
              <Avatar className="h-8 w-8">
                <AvatarFallback className="text-xl bg-accent ">
                  {session?.domain?.alias
                    ? session?.domain?.alias?.[0]
                    : session?.domain?.name?.[0]}
                </AvatarFallback>
              </Avatar>
            </div>
          ) : null}
          <div
            className={cn(
              "transition-all duration-100 inline overflow-hidden border-1",
              "max-w-[10rem]",
              expand ? " opacity-100 " : "opacity-0 w-[0px]",
            )}
          >
            <Button
              variant="outline"
              size="sm"
              className={cn(
                "justify-start transition-all duration-200 delay-150 max-w-full text-popover-foreground",
              )}
              aria-placeholder="Select Domain"
            >
              {session?.domain ? (
                <span className="truncate ...">
                  {session?.domain?.alias
                    ? session?.domain?.alias
                    : session?.domain?.name}
                </span>
              ) : (
                <>
                  <div className="">Select Domain</div>
                </>
              )}
              <ArrowRightLeft className="ml-2 h-4 w-4 shrink-0 opacity-50 text-popover-foreground" />
            </Button>
          </div>
        </div>
      </PopoverTrigger>
      <PopoverContent side={isMobile ? "bottom" : "right"} align="start">
        <div className="flex flex-col ">
          {error ? (
            <div className="bg-red-100 text-red-700 p-4 border border-red-200">
              <div className="flex items-center gap-3">
                <TriangleAlert className="h-10 w-10 text-red-500 mr-2 " />
                <span className="font-medium">Oops, Something went wrong</span>
              </div>
            </div>
          ) : (
            <>
              <div className=" text-center text-muted-foreground mb-2">
                SWITCH DOMAINS
              </div>
              <Separator className="my-2" />
              {domains?.map((domain) => (
                <Button
                  key={domain.id}
                  type={domain.id === session?.domain?.id ? "button" : "submit"}
                  variant="ghost"
                  className={cn(
                    "w-[100%] flex justify-between hover:bg-primary/10 dark:hover:bg-accent",
                    domain.id === session?.domain?.id
                      ? "bg-primary/10 cursor-default"
                      : "transparent",
                  )}
                  onClick={async () => {
                    domain.id !== session?.domain?.id &&
                      (await DomainLogin(domain.id as string));
                  }}
                >
                  <span className="truncate ...">
                    {domain.alias ? domain.alias : domain.name}
                  </span>
                  <ArrowRightLeft className="ml-2 h-4 w-4 shrink-0 opacity-50" />
                </Button>
              ))}
              <Separator className="my-2" />
              <Link
                className={buttonVariants({ variant: "outline" })}
                href="/domains"
              >
                View other domains
              </Link>
            </>
          )}
        </div>
      </PopoverContent>
    </Popover>
  );
}
