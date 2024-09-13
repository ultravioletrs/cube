import { cn } from "@/lib/utils";
import { Skeleton } from "../ui/skeleton";

export function TableSkeleton({ className }: { className?: string }) {
  return (
    <div className="flex flex-col gap-2">
      <div className="flex  items-center py-4 gap-4 sm:gap-8">
        <div className="w-full">
          <Skeleton className="h-10 w-full rounded-md" />
        </div>
        <Skeleton className="h-10 w-28 rounded-md" />
        <Skeleton className="h-10 w-28 rounded-md" />
      </div>
      <div>
        <Skeleton className={cn("h-[70vh] rounded-xl", className)} />
      </div>
    </div>
  );
}
