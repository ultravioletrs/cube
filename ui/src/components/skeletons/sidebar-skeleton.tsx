import { Skeleton } from "@/components/ui/skeleton";

export function SidebarSkeleton() {
  return (
    <nav className="grid items-start gap-2">
      <Skeleton className="w-[11vw] h-[100vh]" />
    </nav>
  );
}
