"use client";

import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

import { createPageUrl } from "@/lib/utils";
import { Separator } from "@radix-ui/react-dropdown-menu";
import { ListFilter } from "lucide-react";
import { usePathname, useSearchParams } from "next/navigation";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { Icons } from "../icons";
import { Badge } from "../ui/badge";
import { Button } from "../ui/button";
import { Checkbox } from "../ui/checkbox";

type Status = "enabled" | "disabled" | "all";

export default function Statusbutton() {
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const [selectedOption, setSelectedOption] = useState<Status>();

  const router = useRouter();
  const handleStatusChange = (status: Status) => {
    setSelectedOption(status);
    router.push(createPageUrl(searchParams, pathname, status, "status"));
  };

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild={true}>
        <Button variant="outline" className="hover:bg-primary/10">
          <ListFilter className="h-4 w-4 mr-2 text-popover-foreground" />
          {selectedOption ? (
            <div className="text-center flex flex-row ">
              <span className="hidden sm:block text-popover-foreground">
                Status {" : "}{" "}
              </span>
              <Badge
                variant="outline"
                className="ml-2 rounded text-popover-foreground"
              >
                {selectedOption}
              </Badge>
            </div>
          ) : (
            <span className="text-popover-foreground">Status</span>
          )}
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="center" className="w-[150px] ">
        <div className="flex items-center space-x-2 hover:bg-primary/10 p-2">
          <Checkbox
            id="enabled"
            className="border-slate-600"
            checked={selectedOption === "enabled"}
            onCheckedChange={() =>
              handleStatusChange(
                selectedOption === "enabled"
                  ? ("all" as Status)
                  : ("enabled" as Status),
              )
            }
          />
          <div className="grid gap-1.5 leading-none">
            <label
              htmlFor="enabled"
              className="text-sm font-sm flex flex-row items-center gap-2 cursor-pointer"
            >
              {" "}
              <Icons.enabled className="h-4 w-4 text-enabledentity" />
              Enabled
            </label>
          </div>
        </div>
        <Separator />
        <div className="flex items-center space-x-2 hover:bg-primary/10 p-2">
          <Checkbox
            id="disabled"
            checked={selectedOption === "disabled"}
            className="border-slate-600"
            onCheckedChange={() =>
              handleStatusChange(
                selectedOption === "disabled"
                  ? ("all" as Status)
                  : ("disabled" as Status),
              )
            }
          />
          <div className="grid gap-1.5 leading-none">
            <label
              htmlFor="disabled"
              className="text-sm font-sm flex flex-row items-center gap-2 cursor-pointer"
            >
              {" "}
              <Icons.disabled className="h-4 w-4 text-disabledentity" />
              Disabled
            </label>
          </div>
        </div>
        <Separator />
        <div className="flex items-center space-x-2 mb-2 hover:bg-primary/10 p-2">
          <Checkbox
            id="all"
            checked={selectedOption === "all"}
            className="border-slate-600"
            onCheckedChange={() =>
              handleStatusChange(
                selectedOption === "all"
                  ? ("enabled" as Status)
                  : ("all" as Status),
              )
            }
          />
          <div className="grid gap-1.5 leading-none">
            <label
              htmlFor="all"
              className="text-sm font-sm flex flex-row items-center gap-2 cursor-pointer"
            >
              {" "}
              <Icons.all color="grey" className="h-4 w-4" />
              All
            </label>
          </div>
        </div>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
