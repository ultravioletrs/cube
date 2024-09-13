"use client";

import { statusSchema } from "@/components/tables/schema";

export function DisplayStatusWithIcon({
  status,
}: {
  status: string;
}) {
  const currentStatus = statusSchema.find((state) => state.value === status);

  if (!currentStatus) {
    return (
      <div className="flex items-center">
        <p className="text-center">Unknown</p>
      </div>
    );
  }

  return (
    <div className="flex items-center">
      {currentStatus.icon && (
        <currentStatus.icon
          className="mr-2 h-4 w-4 text-muted-foreground"
          color={currentStatus.colour}
        />
      )}
      <p className="text-center">{currentStatus.label}</p>
    </div>
  );
}
