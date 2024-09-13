import { Skeleton } from "@/components/ui/skeleton";

export function ViewEntitySkeleton() {
  return (
    <div className="container mx-auto pb-4 md:pb-8">
      <Skeleton className="h-[40vh] w-full rounded-md" />
    </div>
  );
}
