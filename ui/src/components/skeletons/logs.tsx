import { TableSkeleton } from "@/components/skeletons/table";

export function LogsSkeleton() {
  return (
    <div className="container mx-auto pb-4 md:pb-8">
      <TableSkeleton className="h-[60vh]" />
    </div>
  );
}
