import { Skeleton } from "@/components/ui/skeleton";

export default function DomainPageLoading() {
  return (
    <div className="container mx-auto py-10 flex flex-col gap-4">
      <div className="flex justify-between flex-row flex-wrap mb-4 ">
        <h1 className="hidden sm:block text-4xl font-bold">Home Page</h1>
        <div className=" flex flex-col md:flex-row gap-2">
          <div className="w-full">
            <Skeleton className="h-10 w-60 rounded-md" />
          </div>

          <div className="flex flex-row gap-2">
            <Skeleton className="h-10 w-20 rounded-md" />
            <Skeleton className="h-10 w-20 rounded-md" />
          </div>
        </div>
      </div>
      <div className="grid lg:grid-cols-4 gap-4">
        <Skeleton className="h-[200px] rounded-md" />
        <Skeleton className="h-[200px] rounded-md" />
        <Skeleton className="h-[200px] rounded-md" />
        <Skeleton className="h-[200px] rounded-md" />
      </div>
      <div className="grid lg:grid-cols-2 gap-4">
        <Skeleton className="h-[350px] rounded-md" />
        <Skeleton className="h-[350px] rounded-md" />
      </div>
    </div>
  );
}
