import { TableSkeleton } from "@/components/skeletons/table";
import { Skeleton } from "@/components/ui/skeleton";

export function EntitiesPageSkeleton() {
  return (
    <div className="container mx-auto mt-12 pb-4 md:pb-8">
      <div className="flex item-center justify-end gap-2">
        <Skeleton className="h-10 w-32 rounded-md" />
        <Skeleton className="h-10 w-44 rounded-md" />
      </div>
      <TableSkeleton className="h-[60vh]" />
    </div>
  );
}
