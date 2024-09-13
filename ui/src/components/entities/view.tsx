"use client";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { darkDialogTheme, lightDialogTheme } from "@/lib/utils";
import type { Metadata } from "@/types/entities";
import { Themes } from "@/types/entities";
import { useTheme } from "next-themes";
import { useState } from "react";
import { JSONTree } from "react-json-tree";
import { Badge } from "../ui/badge";

export function ViewMetadataDialog({ metadata }: { metadata: Metadata }) {
  if (!metadata) {
    metadata = {};
  }
  const [data, setData] = useState(metadata);
  const [open, setOpen] = useState(false);
  const { resolvedTheme } = useTheme();

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild={true}>
        <Button variant="outline">View Metadata</Button>
      </DialogTrigger>
      <DialogContent className="space-x-2 overflow-y-auto md:max-h-[700px] rounded max-w-[400px] md:max-w-[800px]">
        <DialogHeader>
          <DialogTitle>Metadata</DialogTitle>
        </DialogHeader>
        <JSONTree
          data={data}
          theme={
            resolvedTheme === Themes.MidnightSky
              ? darkDialogTheme
              : lightDialogTheme
          }
          shouldExpandNodeInitially={(_keyName, _data, level) => level < 2}
        />
        <DialogFooter className="flex flex-row justify-end gap-2">
          <DialogClose asChild={true}>
            <Button
              type="button"
              variant="secondary"
              onClick={() => {
                setOpen(false);
                setData(metadata);
              }}
            >
              Close
            </Button>
          </DialogClose>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export function DisplayDescription({ description }: { description: string }) {
  return (
    <div className="flex space-x-2">
      <span className="max-w-[200px] md:max-w-[400px] truncate">
        {description}
      </span>
    </div>
  );
}

export function Tags({ tags }: { tags?: string[] }) {
  if (!tags) {
    return null;
  }
  return (
    <div className="flex space-x-2">
      <span className="max-w-[900px] font-medium">
        {tags.map((tag, index) => (
          <Badge
            // biome-ignore lint/suspicious/noArrayIndexKey: Tags are not unique
            key={index}
            variant="secondary"
            className="rounded-sm px-1 font-normal text-xs mx-1 my-1"
          >
            {tag}
          </Badge>
        ))}
      </span>
    </div>
  );
}
