"use client";
import { DisableEntity, EnableEntity } from "@/lib/entities";
import type { EntityType } from "@/types/entities";
import type { Status } from "@absmach/magistrala-sdk";
import { Switch } from "../ui/switch";

export default function StatusSwitch({
  entity,
  status,
  id,
}: {
  entity: EntityType;
  status: Status;
  id: string;
}) {
  const isEntityEnabled = status === "enabled";

  const handleStatusChange = async (isChecked: boolean) => {
    switch (isChecked) {
      case true:
        return await EnableEntity(id, entity);
      case false:
        return await DisableEntity(id, entity);
    }
  };

  return (
    <div className="w-full space-y-6">
      <Switch
        checked={isEntityEnabled}
        onCheckedChange={(isChecked) => {
          handleStatusChange(isChecked);
        }}
      />
    </div>
  );
}
