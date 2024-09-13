"use client";
import { Button } from "@/components/ui/button";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";

export function TimeWindow() {
  return (
    <Popover>
      <PopoverTrigger>
        <Button variant="outline" type="button">
          Open Time Window
        </Button>
      </PopoverTrigger>
      <PopoverContent>
        <div>This will be a time window</div>
      </PopoverContent>
    </Popover>
  );
}
