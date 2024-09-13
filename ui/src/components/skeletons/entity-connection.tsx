import { TableSkeleton } from "@/components/skeletons/table";
import { Skeleton } from "@/components/ui/skeleton";

export function ViewEntityConnectionsSkeleton() {
  return (
    <div className="container mx-auto pb-4 md:pb-8">
      <div className="flex item-center justify-end">
        <Skeleton className="h-10 w-32 rounded-md" />
      </div>
      <TableSkeleton className="h-[60vh]" />
    </div>
  );
}
