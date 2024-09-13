import { Button } from "@/components/ui/button";
import { AcceptInvitation } from "@/lib/invitations";
import { toast } from "sonner";

export const AcceptInvitationButton = ({ domainId }: { domainId: string }) => {
  async function onSubmit() {
    const toastId = toast("Sonner");
    toast.loading("Accepting Invitation...", {
      id: toastId,
    });
    const result = await AcceptInvitation(domainId);
    if (result.error === null) {
      toast.success("Invitation accepted successfully", {
        id: toastId,
      });
    } else {
      toast.error("Failed to accept invitation.", {
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
      className="bg-green-100 px-1"
    >
      <span className="text-green-600">Accept</span>
    </Button>
  );
};
