"use client";

import {
  AlertDialog,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { DeleteUser } from "@/lib/users";
import { EntityType } from "@/types/entities";
import { toast } from "sonner";
import { Button } from "../ui/button";

type Props = {
  showDeleteDialog: boolean;
  setShowDeleteDialog: (show: boolean) => void;
  entity: EntityType;
  id: string;
};

export default function DeleteEntityAlertDialog({
  showDeleteDialog,
  setShowDeleteDialog,
  entity,
  id,
}: Props) {
  return (
    <AlertDialog open={showDeleteDialog} onOpenChange={setShowDeleteDialog}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Are you sure?</AlertDialogTitle>
          <AlertDialogDescription>
            This will delete the {entity} and it will no longer be accessible.
            This action cannot be undone.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
          <Button
            variant="destructive"
            onClick={async () => {
              setShowDeleteDialog(false);

              const toastId = toast("Sonner");
              toast.loading(`Deleting ${entity} ...`, {
                id: toastId,
              });

              const result = await DeleteEntity(entity, id);
              if (result.error === null) {
                toast.success(`${entity} ${id} deleted successfully`, {
                  id: toastId,
                });
              } else {
                toast.error(`Failed to delete ${entity}: ${result.error}`, {
                  id: toastId,
                });
              }
            }}
          >
            Delete
          </Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

function DeleteEntity(entity: EntityType, id: string) {
  switch (entity) {
    case EntityType.User:
      return DeleteUser(id);
    default:
      throw new Error(`Entity ${entity} not found`);
  }
}
