import { Skeleton } from "@/components/ui/skeleton";

export default function HomeRootLoading() {
  return (
    <div className="flex item-center justify-center">
      <main className="container py-24 flex flex-col bg-accent min-h-screen gap-4">
        <div className="flex flex-col md:flex-row items-center gap-8">
          <div className="w-full">
            <Skeleton className="h-10 w-full rounded-md" />
          </div>
          <div className="flex gap-2 justify-end">
            <Skeleton className="h-10 w-20 rounded-md" />
            <Skeleton className="h-10 w-20 rounded-md" />
          </div>
        </div>
        <div>
          <Skeleton className="h-[70vh] rounded-xl" />
        </div>
      </main>
    </div>
  );
}
