"use client";
import { CopyButton } from "@/components/copy";
import { ViewPermissions } from "@/components/entities/permissions";
import { DisplayStatusWithIcon } from "@/components/entities/status-display-with-icon";
import { UnassignUserFromDomainDialog } from "@/components/entities/user-domain-connections";
import { DataTableColumnHeader } from "@/components/tables/column-header";
import DataTable from "@/components/tables/table";
import { Button } from "@/components/ui/button";
import { Dialog } from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { GetDomainPermissions } from "@/lib/domains";
import type { userSchema } from "@/types/schemas";
import type { UsersPage } from "@absmach/magistrala-sdk";
import type { ColumnDef } from "@tanstack/react-table";
import { MoreHorizontal } from "lucide-react";
import { getSession, useSession } from "next-auth/react";
import { useEffect, useState } from "react";
import type { z } from "zod";

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
    accessorKey: "id",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="ID" />
    ),
    cell: ({ row }) => {
      return (
        <div>
          <span>{row.getValue("id")}</span>
          <CopyButton data={row.getValue("id")} />
        </div>
      );
    },
  },
  {
    accessorKey: "permissions",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Permissions" />
    ),
    cell: ({ row }) => {
      return <ViewPermissions permissions={row.getValue("permissions")} />;
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
    id: "actions",
    cell: ({ row }) => {
      const user = row.original;

      return <Actions user={user} />;
    },
  },
];

export function MembersTable({
  membersPage,
  userId,
  page,
  limit,
}: {
  membersPage: UsersPage;
  userId: string;
  page: number;
  limit: number;
}) {
  return (
    <DataTable
      baseColumns={baseColumns}
      allColumns={allColumns}
      searchPlaceHolder="Search Member"
      currentPage={page}
      total={membersPage.total}
      limit={limit}
      data={membersPage.users}
      clickable={false}
      userId={userId}
      canFilter={false}
      noContentPlaceHolder="No members found. Get started by assigning a member to the domain."
    />
  );
}

function Actions({ user }: { user: User }) {
  const [showUnassignDialog, setShowUnassignDialog] = useState(false);
  const [canEdit, setCanEdit] = useState<boolean>();
  const [canDelete, setCanDelete] = useState<boolean>();
  const { status } = useSession();
  useEffect(() => {
    async function getDomainPermissions() {
      const session = await getSession();
      const domainPermissions = await GetDomainPermissions(
        session?.domain?.id as string,
      );
      domainPermissions?.data?.permissions?.includes("edit") &&
        setCanEdit(true);
      domainPermissions?.data?.permissions?.includes("delete") &&
        setCanDelete(true);
    }
    if (status === "authenticated") {
      getDomainPermissions();
    }
  }, [status]);

  return (
    <>
      {canEdit && (
        <DropdownMenu>
          <DropdownMenuTrigger asChild={true}>
            <Button variant="ghost" className="h-8 w-8 p-0">
              <span className="sr-only">Open menu</span>
              <MoreHorizontal className="h-4 w-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem
              onClick={() => navigator.clipboard.writeText(user.id)}
            >
              Copy ID
            </DropdownMenuItem>

            {canDelete && (
              <>
                <DropdownMenuSeparator />
                <DropdownMenuItem onSelect={() => setShowUnassignDialog(true)}>
                  Remove from Domain
                </DropdownMenuItem>
              </>
            )}
          </DropdownMenuContent>
        </DropdownMenu>
      )}
      <Dialog open={showUnassignDialog} onOpenChange={setShowUnassignDialog}>
        <UnassignUserFromDomainDialog
          userId={user.id}
          setOpen={setShowUnassignDialog}
        />
      </Dialog>
    </>
  );
}
