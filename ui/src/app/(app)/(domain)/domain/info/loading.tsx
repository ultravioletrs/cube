import { Skeleton } from "@/components/ui/skeleton";

export default function Loading() {
  return (
    <div className="container mx-auto mt-4 pb-4 md:pb-8">
      <Skeleton className="h-[70vh] w-full rounded-md" />
    </div>
  );
}
