import { TopbarSkeleton } from "@/components/skeletons/topbar-skeleton";
import { Loader2Icon } from "lucide-react";

export default function RootLoading() {
  return (
    <div className="container mt-96 flex flex-row justify-center">
      <TopbarSkeleton />
      <Loader2Icon className="animate-spin h-20 w-20" />
    </div>
  );
}
