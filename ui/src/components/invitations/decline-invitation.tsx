import { Button } from "@/components/ui/button";
import { DeleteInvitation } from "@/lib/invitations";
import { toast } from "sonner";

export const DeclineInvitationButton = ({
  domainId,
  userId,
}: { domainId: string; userId: string }) => {
  async function onSubmit() {
    const toastId = toast("Sonner");
    toast.loading("Declining Invitation...", {
      id: toastId,
    });
    const result = await DeleteInvitation(domainId, userId);
    if (result.error === null) {
      toast.success("Invitation declined successfully", {
        id: toastId,
      });
    } else {
      toast.error("Failed to decline invitation.", {
        id: toastId,
      });
    }
  }
  return (
    <Button
      variant="outline"
      size="sm"
      type="button"
      onClick={onSubmit}
      className="bg-red-100 px-1"
    >
      <span className="text-rose-600">Decline</span>
    </Button>
  );
};
