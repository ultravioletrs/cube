import { Skeleton } from "@/components/ui/skeleton";

export default function Loading() {
  return (
    <div className="container mx-auto mt-4 pb-4 md:pb-8">
      <div className="flex item-center justify-end mt-12 mb-4">
        <Skeleton className="w-40 h-10 rounded-md" />
      </div>
      <div className="flex flex-col gap-2 ">
        <div className="flex items-center justify-end">
          <Skeleton className="h-10 w-20 rounded-md" />
        </div>

        <div>
          <Skeleton className="h-[70vh] rounded-xl" />
        </div>
      </div>
    </div>
  );
}
