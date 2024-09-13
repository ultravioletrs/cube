import { TableSkeleton } from "@/components/skeletons/table";
import { Skeleton } from "@/components/ui/skeleton";

export default function Loading() {
  return (
    <>
      <div className="flex item-center justify-end gap-2">
        <Skeleton className="h-10 w-32 rounded-md" />
        <Skeleton className="h-10 w-44 rounded-md" />
      </div>
      <TableSkeleton className="h-[60vh]" />
    </>
  );
}
