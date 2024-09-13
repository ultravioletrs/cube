"use client";
import { Action, DisplayTimeCell } from "@/components/display-time";
import { DataTableColumnHeader } from "@/components/tables/column-header";
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
import { DeleteInvitation } from "@/lib/invitations";
import type { Invitation, InvitationsPage } from "@absmach/magistrala-sdk";
import type { ColumnDef } from "@tanstack/react-table";
import { Trash } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { toast } from "sonner";
import DataTable from "./data-table";

const baseColumns: ColumnDef<Invitation>[] = [
  {
    accessorKey: "user_id",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="User Name" />
    ),
    cell: ({ row }) => {
      return (
        <div className="flex flex-col">
          {typeof row.original.user_id === "object" ? (
            <>
              <p>{row.original.user_id.name || ""}</p>
            </>
          ) : (
            <p>{row.original.user_id}</p>
          )}
        </div>
      );
    },
  },
  {
    accessorKey: "relation",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Relation" />
    ),
  },
  {
    accessorKey: "invited_by",
    header: "Invited By",
    cell: ({ row }) => {
      return (
        <div className="flex flex-col ">
          {typeof row.original.invited_by === "object" ? (
            <>
              <p>{row.original.invited_by.name || ""}</p>
            </>
          ) : (
            <p>{row.original.invited_by}</p>
          )}
        </div>
      );
    },
  },
  {
    id: "actions",
    cell: ({ row }) => {
      const invitation = row.original;
      return (
        <div className="flex gap-2">
          <Delete
            domainId={
              typeof invitation.domain_id === "string"
                ? invitation.domain_id
                : (invitation.domain_id?.id as string)
            }
            userId={
              typeof invitation.user_id === "string"
                ? invitation.user_id
                : (invitation.user_id?.id as string)
            }
          />
        </div>
      );
    },
  },
];
const allColumns: ColumnDef<Invitation>[] = [
  {
    accessorKey: "user_id",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="User Name" />
    ),
    cell: ({ row }) => {
      return (
        <div className="flex flex-col">
          {typeof row.original.user_id === "object" ? (
            <>
              <p>{row.original.user_id.name || ""}</p>
            </>
          ) : (
            <p>{row.original.user_id}</p>
          )}
        </div>
      );
    },
  },
  {
    accessorKey: "relation",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Relation" />
    ),
  },
  {
    accessorKey: "invited_by",
    header: "Invited By",
    cell: ({ row }) => {
      return (
        <div className="flex flex-col ">
          {typeof row.original.invited_by === "object" ? (
            <>
              <p>{row.original.invited_by.name || ""}</p>
            </>
          ) : (
            <p>{row.original.invited_by}</p>
          )}
        </div>
      );
    },
  },
  {
    accessorKey: "confirmed_at",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Confirmed At" />
    ),
    cell: ({ row }) => {
      return (
        <DisplayTimeCell
          time={row.getValue("confirmed_at")}
          action={Action.Confirmed}
        />
      );
    },
  },
  {
    id: "actions",
    cell: ({ row }) => {
      const invitation = row.original;
      return (
        <div className="flex gap-2">
          <Delete
            domainId={
              typeof invitation.domain_id === "string"
                ? invitation.domain_id
                : (invitation.domain_id?.id as string)
            }
            userId={
              typeof invitation.user_id === "string"
                ? invitation.user_id
                : (invitation.user_id?.id as string)
            }
          />
        </div>
      );
    },
  },
];

export function InvitationsTable({
  invitationsPage,
  page,
  limit,
}: {
  invitationsPage: InvitationsPage;
  page: number;
  limit: number;
}) {
  const [columns, setColumns] = useState<ColumnDef<Invitation>[]>(allColumns);
  const handleResize = useCallback(() => {
    const isSmallScreen = () => window.innerWidth < 768;
    setColumns(isSmallScreen() ? baseColumns : allColumns);
  }, []);

  useEffect(() => {
    if (columns) {
      handleResize();
      window.addEventListener("resize", handleResize);
      return () => window.removeEventListener("resize", handleResize);
    }
  }, [handleResize, columns]);

  return (
    <DataTable
      columns={columns}
      currentPage={page}
      total={invitationsPage.total}
      limit={limit}
      data={invitationsPage.invitations}
    />
  );
}

function Delete({ domainId, userId }: { domainId: string; userId: string }) {
  const handleDelete = async () => {
    const toastId = toast("Sonner");
    toast.loading("Deleting Invitation...", { id: toastId });

    const result = await DeleteInvitation(domainId, userId);

    if (result.error === null) {
      toast.success("Invitation deleted successfully", { id: toastId });
    } else {
      toast.error(`Failed to delete invitation: ${result.error}`, {
        id: toastId,
      });
    }
  };
  return (
    <Dialog>
      <DialogTrigger asChild={true}>
        <Button variant="destructive" size="icon">
          <Trash className="h-4 w-4" />
        </Button>
      </DialogTrigger>
      <DialogContent className="rounded max-w-[400px] sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle>Delete Invitation</DialogTitle>
        </DialogHeader>
        <p>
          Are you sure you want to
          <span className="text-red-600 font-bold"> delete </span>
          the invitation?
        </p>
        <DialogFooter className="flex flex-row justify-end gap-2">
          <DialogClose asChild={true}>
            <Button type="button" variant="secondary">
              Close
            </Button>
          </DialogClose>
          <Button type="button" variant="destructive" onClick={handleDelete}>
            Delete
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
