"use client";

import { InviteMultipleUsersDialog } from "@/components/entities/user-domain-connections";
import { Button } from "@/components/ui/button";
import { Dialog, DialogTrigger } from "@/components/ui/dialog";
import { Plus } from "lucide-react";
import { useState } from "react";

export const SendInvitationForm = ({
  id,
}: {
  id: string;
}) => {
  const [open, setOpen] = useState(false);
  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild={true}>
        <Button>
          <Plus className="h-5 mr-2" />
          <span>Send Invitation</span>
        </Button>
      </DialogTrigger>
      <InviteMultipleUsersDialog id={id} setOpen={setOpen} />
    </Dialog>
  );
};
