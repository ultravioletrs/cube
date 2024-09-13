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
import { Button } from "@/components/ui/button";
import { DisableDomain, EnableDomain } from "@/lib/domains";
import { toast } from "sonner";

type DomainStatusDialogProps = {
  showStatusDialog: boolean;
  setShowStatusDialog: (show: boolean) => void;
  isEnabled: boolean;
  name: string;
  id: string;
};
export function DomainStatusDialog({
  showStatusDialog,
  isEnabled,
  setShowStatusDialog,
  name,
  id,
}: DomainStatusDialogProps) {
  return (
    <AlertDialog open={showStatusDialog} onOpenChange={setShowStatusDialog}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Are you sure?</AlertDialogTitle>
          <AlertDialogDescription>
            Are you sure you want to
            <span
              className={`font-bold ${
                isEnabled ? "text-red-500" : "text-green-500"
              }`}
            >
              {isEnabled ? " disable" : " enable"}
            </span>
            <span className="overflow-auto">{` ${name}?`}</span>
            {isEnabled
              ? " Disabling the domain will revoke the access for users who are not domain admins."
              : " Enabling the domain will make it accessible to all other members of the domain."}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
          <Button
            variant={isEnabled ? "destructive" : "default"}
            onClick={async () => {
              const toastId = toast("Sonner");
              switch (isEnabled) {
                case true: {
                  toast.loading("Disabling domain ...", {
                    id: toastId,
                  });

                  const response = await DisableDomain(id);
                  if (response.error === null) {
                    toast.success(`Domain ${name} disabled successfully`, {
                      id: toastId,
                    });
                    setShowStatusDialog(false);
                  } else {
                    toast.error(
                      `Failed to disable domain ${name}: ${response.error}`,
                      {
                        id: toastId,
                      },
                    );
                  }
                  break;
                }

                case false: {
                  toast.loading("Enabling domain ...", {
                    id: toastId,
                  });

                  const response = await EnableDomain(id);
                  if (response.error === null) {
                    toast.success(`Domain ${name} enabled successfully`, {
                      id: toastId,
                    });
                    setShowStatusDialog(false);
                  } else {
                    toast.error(
                      `Failed to enable domain ${name}: ${response.error}`,
                      {
                        id: toastId,
                      },
                    );
                  }
                  break;
                }
              }
            }}
          >
            {isEnabled ? "Disable" : "Enable"}
          </Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
