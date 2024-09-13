import { Skeleton } from "@/components/ui/skeleton";

const skeletonCards: number[] = [1, 2, 3, 4, 5, 6, 7, 8, 9];

export default function CardsSkeleton() {
  return (
    <div className="grid grid-flow-row grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-4 w-full content-center justify-items-center p-4 rounded-2xl bg-accent">
      {skeletonCards.map((num) => (
        <Skeleton
          key={num}
          className="w-[200px] h-[300px] rounded-xl bg-card"
        />
      ))}
    </div>
  );
}
