import { TableSkeleton } from "@/components/skeletons/table";

export default function Loading() {
  return (
    <div className="container mx-auto mt-4 pb-4 md:pb-8">
      <TableSkeleton className="h-[60vh]" />
    </div>
  );
}
