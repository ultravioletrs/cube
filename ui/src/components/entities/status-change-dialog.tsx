import { Button } from "@/components/ui/button";
import {
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { DisableEntity, EnableEntity } from "@/lib/entities";
import type { EntityType } from "@/types/entities";
import type { User } from "@absmach/magistrala-sdk";
import type { Dispatch, SetStateAction } from "react";
import { toast } from "sonner";

export const EntityStatusChangeDialog = ({
  entity,
  setOpen,
  entityType,
}: {
  entity: User;
  setOpen: Dispatch<SetStateAction<boolean>>;
  entityType: EntityType;
}) => {
  const isEnabled = entity.status === "enabled";
  const action = isEnabled ? DisableEntity : EnableEntity;
  const handleAction = async () => {
    const toastId = toast("Sonner");
    toast.loading(`${isEnabled ? "Disabling" : "Enabling"} ${entity.name}...`, {
      id: toastId,
    });

    const result = await action(entity.id as string, entityType);

    if (result.error === null) {
      toast.success(
        `${entity.name} ${isEnabled ? "disabled" : "enabled"} successfully.`,
        {
          id: toastId,
        },
      );
      setOpen(false);
    } else {
      toast.error(
        `Failed to ${isEnabled ? "disable" : "enable"} ${entity.name}: ${
          result.error
        }`,
        {
          id: toastId,
        },
      );
    }
  };
  return (
    <DialogContent className="rounded max-w-[400px] sm:max-w-[425px]">
      <DialogHeader>
        <DialogTitle>
          {isEnabled ? "Disable Entity" : "Enable Entity"}
        </DialogTitle>
      </DialogHeader>
      <DialogDescription>
        Are you sure you want to
        <span
          className={`font-bold ${
            entity.status === "enabled" ? "text-red-500" : "text-green-500"
          }`}
        >
          {isEnabled ? " disable" : " enable"}
        </span>
        <span className="overflow-auto">{` ${entity.name}?`}</span>
      </DialogDescription>
      <DialogFooter>
        <DialogClose asChild={true}>
          <Button type="button" variant="secondary">
            Close
          </Button>
        </DialogClose>
        <Button type="button" onClick={handleAction}>
          {isEnabled ? "Disable" : "Enable"}
        </Button>
      </DialogFooter>
    </DialogContent>
  );
};
