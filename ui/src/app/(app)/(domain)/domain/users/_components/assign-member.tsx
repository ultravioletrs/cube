"use client";
import { AssignMultipleUsersToDomainDialog } from "@/components/entities/user-domain-connections";
import { Button } from "@/components/ui/button";
import { Dialog, DialogTrigger } from "@/components/ui/dialog";
import { useState } from "react";

export function AssignMember({ domainId }: { domainId: string }) {
  const [open, setOpen] = useState(false);
  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild={true}>
        <Button>Assign User</Button>
      </DialogTrigger>

      <AssignMultipleUsersToDomainDialog
        domainId={domainId}
        setOpen={setOpen}
      />
    </Dialog>
  );
}
