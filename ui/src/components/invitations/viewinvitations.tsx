"use client";

import { Icons } from "@/components/icons";
import { Button, buttonVariants } from "@/components/ui/button";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { ScrollArea } from "@/components/ui/scroll-area";
import { toSentenseCase } from "@/lib/utils";
import { Themes } from "@/types/entities";
import type {
  DomainBasicInfo,
  Invitation,
  InvitationsPage,
  UserBasicInfo,
} from "@absmach/magistrala-sdk";
import { TriangleAlert } from "lucide-react";
import { useTheme } from "next-themes";
import Link from "next/link";
import { useState } from "react";
import { AcceptInvitationButton } from "./accept-invitation";
import { DeclineInvitationButton } from "./decline-invitation";

export function ViewUserInvitations({
  invitations,
  error,
}: {
  invitations: InvitationsPage;
  error: string | null;
}) {
  const [open, setOpen] = useState(false);
  const { resolvedTheme } = useTheme();

  const hasInvitations = invitations?.invitations?.length > 0;
  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild={true}>
        <Button variant="ghost" className="relative text-popover-foreground">
          {hasInvitations && (
            <span
              className={`absolute top-0 right-0 block h-5 w-5 rounded-full bg-gray-300 border border-white ${
                resolvedTheme === Themes.MidnightSky ? "text-black" : ""
              }`}
            >
              {invitations.total}
            </span>
          )}
          <Icons.inbox className="h-6 w-6" />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-96 me-4 mt-4 p-2">
        <h1 className="text-center text-xl font-semibold mb-2">Invitations</h1>
        <ScrollArea className="h-96 p-2">
          <div className="flex flex-col gap-2">
            {error ? (
              <div className="bg-red-100 text-red-700 p-4 border border-red-200 flex flex-col items-center justify-center h-full w-full mt-20">
                <TriangleAlert className="h-12 w-12 text-red-500 mr-2" />
                <p className="font-medium mb-4 mt-4">Oops!</p>
                <p className="font-medium ">Something went wrong</p>
              </div>
            ) : invitations?.invitations?.length ? (
              <div className="flex flex-col gap-2">
                <h4 className="pb-2 text-center">
                  You have {invitations?.total} pending invitations
                </h4>
                {invitations?.invitations?.map((invitation: Invitation) => {
                  const invitedBy =
                    typeof invitation.invited_by === "string"
                      ? invitation.invited_by
                      : ((invitation.invited_by as UserBasicInfo)
                          .name as string);
                  const domain =
                    typeof invitation.domain_id === "string"
                      ? invitation.domain_id
                      : ((invitation.domain_id as DomainBasicInfo)
                          .name as string);

                  return (
                    <div
                      key={
                        (invitation.domain_id as string) + invitation.user_id
                      }
                      className=" border-b-4 grid gap-1 p-2 rounded"
                    >
                      <div className="grid grid-cols-[40px_1fr_2fr] items-center ">
                        <div className="flex justify-start items-center gap-2">
                          <Icons.domain className="h-8 w-8" />
                        </div>
                        <div className="space-y-1">
                          <p className="font-medium max-w-[150px] truncate">
                            {toSentenseCase(domain)}
                          </p>
                          <p className="text-sm text-gray-500 dark:text-gray-400 max-w-[160px] truncate">
                            Invited by {toSentenseCase(invitedBy)}
                          </p>
                        </div>
                        <div className="space-y-1 text-right">
                          <p className="font-medium text-sm">
                            {toSentenseCase(invitation.relation as string)}
                          </p>
                          <p className="text-sm text-gray-500 dark:text-gray-400">
                            {invitation.created_at
                              ? new Intl.DateTimeFormat("en-GB", {
                                  day: "numeric",
                                  month: "long",
                                  year: "numeric",
                                }).format(new Date(invitation.created_at))
                              : "N/A"}
                          </p>
                        </div>
                      </div>
                      <div className="col-span-3 flex justify-center gap-6 mt-1">
                        <AcceptInvitationButton
                          domainId={
                            typeof invitation.domain_id === "string"
                              ? invitation.domain_id
                              : (invitation.domain_id?.id as string)
                          }
                        />
                        <DeclineInvitationButton
                          domainId={
                            typeof invitation.domain_id === "string"
                              ? invitation.domain_id
                              : (invitation.domain_id?.id as string)
                          }
                          userId={
                            typeof invitation.user_id === "string"
                              ? invitation.user_id
                              : (invitation.user_id?.id as string)
                          }
                        />
                      </div>
                    </div>
                  );
                })}

                <Link
                  className={buttonVariants({ variant: "ghost" })}
                  href="/invitations"
                  onClick={() => setOpen(false)}
                >
                  View All
                </Link>
              </div>
            ) : (
              <p className="text-center">No Invitations</p>
            )}
          </div>
        </ScrollArea>
      </PopoverContent>
    </Popover>
  );
}
