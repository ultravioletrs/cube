"use client";
import { DisplayTimeCell } from "@/components/display-time";
import DeleteEntityAlertDialog from "@/components/entities/delete";
import { EntityStatusChangeDialog } from "@/components/entities/status-change-dialog";
import { DisplayStatusWithIcon } from "@/components/entities/status-display-with-icon";
import { DataTableColumnHeader } from "@/components/tables/column-header";
import { DisplayTags } from "@/components/tables/table";
import DataTable from "@/components/tables/table";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { type EntityFetchData, FetchData } from "@/lib/actions";
import { EntityType } from "@/types/entities";
import type { userSchema } from "@/types/schemas";
import type { UsersPage } from "@absmach/magistrala-sdk";
import { Dialog } from "@radix-ui/react-dialog";
import type { ColumnDef } from "@tanstack/react-table";
import { MoreHorizontal } from "lucide-react";
import Link from "next/link";
import { useEffect, useState } from "react";
import type { z } from "zod";
import { InviteUser } from "../[id]/_components/update-user";
import { AssignDomain } from "../[id]/_components/update-user";

type User = z.infer<typeof userSchema>;

const baseColumns: ColumnDef<User>[] = [
  {
    accessorKey: "name",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Name" />
    ),
  },
  {
    accessorKey: "status",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Status" />
    ),
    cell: ({ row }) => {
      return <DisplayStatusWithIcon status={row.getValue("status")} />;
    },
  },
  {
    id: "actions",
    cell: ({ row }) => {
      const user = row.original;

      return <Actions user={user} />;
    },
  },
];
const allColumns: ColumnDef<User>[] = [
  {
    accessorKey: "name",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Name" />
    ),
  },
  {
    accessorKey: "tags",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Tags" />
    ),
    cell: ({ row }) => {
      return <DisplayTags tags={row.getValue("tags")} />;
    },
  },
  {
    accessorKey: "status",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Status" />
    ),
    cell: ({ row }) => {
      return <DisplayStatusWithIcon status={row.getValue("status")} />;
    },
  },
  {
    accessorKey: "created_at",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Created At" />
    ),
    cell: ({ row }) => {
      return <DisplayTimeCell time={row.getValue("created_at")} />;
    },
  },
  {
    id: "actions",
    cell: ({ row }) => {
      const user = row.original;

      return <Actions user={user} />;
    },
  },
];

export function UsersTable({
  usersPage,
  page,
  limit,
}: {
  usersPage: UsersPage;
  page: number;
  limit: number;
}) {
  return (
    <DataTable
      baseColumns={baseColumns}
      allColumns={allColumns}
      searchPlaceHolder="Search User"
      currentPage={page}
      total={usersPage.total}
      limit={limit}
      data={usersPage.users}
      href="/platform-management/users"
      filterByIdentity={true}
      hasTags={true}
      noContentPlaceHolder="No users found. Get started by creating one."
    />
  );
}

function Actions({ user }: { user: User }) {
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);
  const [showInviteUserDialog, setShowInviteUserDialog] = useState(false);
  const [showAssignDomainDialog, setShowAssignDomainDialog] = useState(false);
  const [showStatusChangeDialog, setShowStatusChangeDialog] = useState(false);
  const [initData, setInitData] = useState<EntityFetchData | null>(null);
  useEffect(() => {
    async function getData() {
      let data: EntityFetchData | null = null;
      const pgm = {
        offset: 0,
        limit: 20,
      };
      switch (true) {
        case showInviteUserDialog: {
          data = await FetchData(EntityType.Domain, pgm);
          break;
        }
        case showAssignDomainDialog: {
          data = await FetchData(EntityType.Domain, pgm);
          break;
        }
      }
      setInitData(data);
    }
    getData();
  }, [showInviteUserDialog, showAssignDomainDialog]);
  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger asChild={true}>
          <Button variant="ghost" className="h-8 w-8 p-0">
            <span className="sr-only">Open menu</span>
            <MoreHorizontal className="h-4 w-4" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuItem asChild={true}>
            <Link href={`/platform-management/users/${user.id}`}>View</Link>
          </DropdownMenuItem>
          <DropdownMenuItem
            onClick={() => navigator.clipboard.writeText(user.id)}
          >
            Copy ID
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem onSelect={() => setShowAssignDomainDialog(true)}>
            Assign to domain
          </DropdownMenuItem>
          <DropdownMenuItem onSelect={() => setShowInviteUserDialog(true)}>
            Invite to domain
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem onSelect={() => setShowStatusChangeDialog(true)}>
            {user.status === "enabled" ? "Disable" : "Enable"}
          </DropdownMenuItem>
          <DropdownMenuItem
            onSelect={() => setShowDeleteDialog(true)}
            className="text-red-600 focus:text-red-600"
          >
            Delete
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
      <DeleteEntityAlertDialog
        showDeleteDialog={showDeleteDialog}
        setShowDeleteDialog={setShowDeleteDialog}
        entity={EntityType.User}
        id={user.id}
      />
      <InviteUser
        showInviteUserDialog={showInviteUserDialog}
        initData={initData as EntityFetchData}
        setShowInviteUserDialog={setShowInviteUserDialog}
        userId={user.id}
      />
      <AssignDomain
        showAssignDomainDialog={showAssignDomainDialog}
        initData={initData as EntityFetchData}
        setShowAssignDomainDialog={setShowAssignDomainDialog}
        userId={user.id}
      />
      <Dialog
        open={showStatusChangeDialog}
        onOpenChange={setShowStatusChangeDialog}
      >
        <EntityStatusChangeDialog
          entity={user}
          setOpen={setShowStatusChangeDialog}
          entityType={EntityType.User}
        />
      </Dialog>
    </>
  );
}
