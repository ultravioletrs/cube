import { TableSkeleton } from "@/components/skeletons/table";
import { Skeleton } from "@/components/ui/skeleton";

export default function Loading() {
  return (
    <main className="w-full sm:container bg-accent/10 pt-20 p-8 md:p-20">
      <div className="flex item-center justify-end gap-2">
        <Skeleton className="h-10 w-32 rounded-md" />
      </div>
      <TableSkeleton className="h-[70vh]" />
    </main>
  );
}
