import { Trash } from "lucide-react";
import type { Dispatch, SetStateAction } from "react";
import { Button } from "../ui/button";

export default function ClearDateTime({
  setDate,
}: { setDate: Dispatch<SetStateAction<Date | undefined>> }) {
  return (
    <Button
      type="button"
      variant="outline"
      size="icon"
      onClick={() => setDate(undefined)}
    >
      <Trash className="h-4 w-4" />
    </Button>
  );
}
