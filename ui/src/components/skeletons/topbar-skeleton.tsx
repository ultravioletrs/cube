import { Skeleton } from "@/components/ui/skeleton";

export function TopbarSkeleton() {
  return (
    <div className="fixed top-0 left-0 right-0 supports-backdrop-blur:bg-background/60 border-b bg-background/95 backdrop-blur z-20">
      <nav className="h-14 flex items-center justify-between px-4">
        <Skeleton className="w-40 h-10 rounded-md" />
        <div className="flex flex-row items-center gap-4">
          <Skeleton className="w-10 h-10 rounded-md" />
          <Skeleton className="w-7 h-7 rounded-full" />
          <Skeleton className="w-10 h-10 rounded-md" />
        </div>
      </nav>
    </div>
  );
}
