import ClearDateTime from "@/components/time-picker/clear-date-time";
import { TimePicker } from "@/components/time-picker/time-picker";
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
import { CalendarIcon } from "lucide-react";
import type { Dispatch, SetStateAction } from "react";

export const DateTimeSelector = ({
  time,
  setTime,
}: {
  time: Date | undefined;
  setTime: Dispatch<SetStateAction<Date | undefined>>;
}) => (
  <Popover>
    <div className="flex flex-row">
      <PopoverTrigger asChild={true}>
        <FormControl className="w-full mr-2">
          <Button
            variant="outline"
            className={cn(
              "text-left font-normal",
              !time && "text-muted-foreground",
            )}
          >
            {time ? format(time, "PPP HH:mm:ss") : <span>Pick a date</span>}
            <CalendarIcon className="ml-auto h-4 w-4 opacity-50" />
          </Button>
        </FormControl>
      </PopoverTrigger>
      <PopoverContent className="w-auto p-0" align="start">
        <Calendar
          mode="single"
          selected={time}
          onSelect={setTime}
          initialFocus={true}
        />
        <div className="p-3 border-t border-border">
          <TimePicker setDate={setTime} date={time} />
        </div>
      </PopoverContent>
      <ClearDateTime setDate={setTime} />
    </div>
  </Popover>
);
