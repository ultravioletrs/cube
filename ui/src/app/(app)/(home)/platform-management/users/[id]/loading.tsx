import { Skeleton } from "@/components/ui/skeleton";

export default function Loading() {
  return (
    <div className="grid grid-rows-2 gap-4 mt-8">
      <Skeleton className=" h-[450px] w-full rounded-md" />
      <Skeleton className="w-full rounded-md" />
    </div>
  );
}
