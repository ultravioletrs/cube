"use client";
import { Button } from "@/components/ui/button";
import { Calendar } from "@/components/ui/calendar";
import { FormControl } from "@/components/ui/form";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { cn } from "@/lib/utils";
import { format } from "date-fns";
import { Calendar as CalendarIcon } from "lucide-react";
import type { ControllerRenderProps } from "react-hook-form";
import { TimePicker } from "./time-picker";
// biome-ignore lint/suspicious/noExplicitAny: The DateTimePicker is meant to be used in multiple forms with any data type.
type DateTimePickerController = ControllerRenderProps<any>;

type Props = {
  field: DateTimePickerController;
};

export function DateTimePicker({ field }: Props) {
  return (
    <Popover>
      <FormControl>
        <PopoverTrigger asChild={true}>
          <Button
            variant="outline"
            className={cn(
              "text-left font-normal",
              !field.value && "text-muted-foreground",
            )}
          >
            {field.value ? (
              format(field.value, "PPP HH:mm:ss")
            ) : (
              <span>Pick a date</span>
            )}
            <CalendarIcon className="ml-auto h-4 w-4 opacity-50" />
          </Button>
        </PopoverTrigger>
      </FormControl>
      <PopoverContent className="w-auto p-0" align="start">
        <Calendar
          mode="single"
          selected={field.value}
          onSelect={field.onChange}
          initialFocus={true}
        />
        <div className="p-3 border-t border-border">
          <TimePicker setDate={field.onChange} date={field.value} />
        </div>
      </PopoverContent>
    </Popover>
  );
}
