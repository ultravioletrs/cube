"use client";
import {
  AssignUserToMultipleDomainsDialog,
  InviteUserToMultipleDomainsDialog,
} from "@/components/entities/user-domain-connections";
import { Icons } from "@/components/icons";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { TableCell, TableHead, TableRow } from "@/components/ui/table";
import type { EntityFetchData } from "@/lib/actions";
import { UpdateUserIdentity, UpdateUserRole } from "@/lib/users";
import type { User } from "@absmach/magistrala-sdk";
import { zodResolver } from "@hookform/resolvers/zod";
import { Pencil } from "lucide-react";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";
import { z } from "zod";

export function UpdateEmail({ email, id }: { email: string; id: string }) {
  const [isEditing, setIsEditing] = useState(false);

  const formSchema = z.object({
    email: z.string().email({ message: "Invalid email address" }),
  });
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      email: email,
    },
  });

  async function onSubmit(values: z.infer<typeof formSchema>) {
    const user: User = {
      id: id,
      credentials: {
        identity: values.email,
      },
    };

    const toastId = toast("Sonner");
    toast.loading("Updating user ...", {
      id: toastId,
    });

    const result = await UpdateUserIdentity(user);
    if (result.error === null) {
      toast.success(`User ${result.data} updated successfully`, {
        id: toastId,
      });
      setIsEditing(false);
    } else {
      toast.error(`Failed to update user: ${result.error}`, {
        id: toastId,
      });
    }
  }

  return (
    <TableRow>
      <TableHead>Email</TableHead>

      <TableCell>
        <Form {...form}>
          <form
            onSubmit={form.handleSubmit(onSubmit)}
            className=" flex flex-row justify-between"
          >
            <FormField
              control={form.control}
              name="email"
              defaultValue={email}
              render={({ field }) => (
                <FormItem className="w-full">
                  <FormControl className="w-4/5">
                    <Input {...field} disabled={!isEditing} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <div className="flex flex-row gap-2">
              {isEditing ? (
                <div className="flex flex-row gap-2">
                  <Button type="submit" variant="outline" size="icon">
                    <Icons.approve className="h-5 w-5" />
                  </Button>
                  <Button
                    type="button"
                    variant="outline"
                    size="icon"
                    onClick={() => {
                      form.reset();
                      setIsEditing(false);
                    }}
                  >
                    <Icons.discard className="h-5 w-5" />
                  </Button>
                </div>
              ) : (
                <Button
                  type="button"
                  variant="outline"
                  size="icon"
                  onClick={() => setIsEditing(true)}
                  className="hover:bg-primary/10"
                >
                  <Pencil className="h-4 w-4" />
                </Button>
              )}
            </div>
          </form>
        </Form>
      </TableCell>
    </TableRow>
  );
}

export function UpdateRole({
  role,
  id,
  name,
}: {
  role: string;
  id: string;
  name: string;
}) {
  const [open, setOpen] = useState(false);

  async function onSubmit(formData: FormData) {
    const toastId = toast("Sonner");
    toast.loading("Updating user ...", {
      id: toastId,
    });

    const result = await UpdateUserRole(formData);
    if (result.error === null) {
      toast.success(`User ${result.data} updated successfully`, {
        id: toastId,
      });
      setOpen(false);
    } else {
      toast.error(`Failed to update user: ${result.error}`, {
        id: toastId,
      });
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild={true}>
        <Button
          type="button"
          variant="outline"
          size="icon"
          className="hover:bg-primary/10"
        >
          <Pencil className="h-4 w-4" />
        </Button>
      </DialogTrigger>
      <DialogContent className="rounded max-w-[400px] sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle>Update Role</DialogTitle>
        </DialogHeader>
        <form
          action={async (formData) => {
            await onSubmit(formData);
          }}
        >
          <div>
            Are you sure you want to update the role of
            <Badge className="rounded-sm px-1 font-normal text-xs mx-1">
              {name}
            </Badge>
            to
            <Role role={role} />
          </div>
          <DialogFooter className="flex flex-row justify-end gap-2">
            <DialogClose asChild={true}>
              <Button type="button" variant="secondary">
                Close
              </Button>
            </DialogClose>

            <input type="hidden" name="entityId" value={id} />
            <Button variant="outline" type="submit">
              Update
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

function Role({ role }: { role: string }) {
  if (role === "admin") {
    return (
      <>
        <input type="hidden" name="role" value="user" />
        <span className="font-bold"> user.</span>
        <br />
        This will revoke all super admin priviledges from the user and make the
        user a normal user.
      </>
    );
  }
  return (
    <>
      <input type="hidden" name="role" value="admin" />
      <span className="font-bold"> admin.</span>
      <br />
      This will make the user a super admin in the system.
    </>
  );
}

export const AssignDomain = ({
  showAssignDomainDialog,
  initData,
  setShowAssignDomainDialog,
  userId,
}: {
  showAssignDomainDialog: boolean;
  initData: EntityFetchData;
  setShowAssignDomainDialog: (show: boolean) => void;
  userId: string;
}) => {
  return (
    <Dialog
      open={showAssignDomainDialog}
      onOpenChange={setShowAssignDomainDialog}
    >
      <AssignUserToMultipleDomainsDialog
        initData={initData as EntityFetchData}
        setOpen={setShowAssignDomainDialog}
        userId={userId}
      />
    </Dialog>
  );
};

export const InviteUser = ({
  showInviteUserDialog,
  initData,
  setShowInviteUserDialog,
  userId,
}: {
  showInviteUserDialog: boolean;
  initData: EntityFetchData;
  setShowInviteUserDialog: (show: boolean) => void;
  userId: string;
}) => {
  return (
    <Dialog open={showInviteUserDialog} onOpenChange={setShowInviteUserDialog}>
      <InviteUserToMultipleDomainsDialog
        id={userId}
        initData={initData}
        setOpen={setShowInviteUserDialog}
      />
    </Dialog>
  );
};
