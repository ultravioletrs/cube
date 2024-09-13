"use client";

import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";

import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { DomainLogin } from "@/lib/actions";
import type { Domain } from "@absmach/magistrala-sdk";
import { DisplayStatusWithIcon } from "../../../../components/entities/status-display-with-icon";

export function DomainCard({ domain }: { domain: Domain }) {
  return (
    <button
      type="button"
      className="pt-2"
      onClick={async () => {
        await DomainLogin(domain.id as string);
      }}
    >
      <Card className="w-[200px] h-[300px] flex flex-col items-center justify-items-stretch pt-4 px-4 gap-2  hover:scale-110 duration-75 hover:bg-cardhover transform-gpu ">
        <CardHeader className="p-0 flex flex-col justify-between max-w-full truncate ...">
          <CardTitle className="truncate ... p-1">{domain.name}</CardTitle>
          <CardDescription>{domain.permission}</CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col gap-4 justify-start max-w-full">
          <Avatar className="h-20 w-20 self-center p-0">
            <AvatarFallback className="text-3xl bg-accent">
              {domain?.name?.[0]}
            </AvatarFallback>
          </Avatar>
          <CardDescription>
            <DisplayStatusWithIcon status={domain?.status as string} />
          </CardDescription>
        </CardContent>
        <CardFooter className="self-center max-w-full justify-self-end p-0">
          <CardDescription className=" max-w-full break-words">
            {domain.alias}
          </CardDescription>
        </CardFooter>
      </Card>
    </button>
  );
}
