"use client";

import DataTable from "@/app/(app)/(domain)/domain/invitations/_components/data-table";
import { Action, DisplayTimeCell } from "@/components/display-time";
import { AcceptInvitationButton } from "@/components/invitations/accept-invitation";
import { DeclineInvitationButton } from "@/components/invitations/decline-invitation";
import { DataTableColumnHeader } from "@/components/tables/column-header";
import { Badge } from "@/components/ui/badge";
import type { Invitation, InvitationsPage } from "@absmach/magistrala-sdk";
import type { ColumnDef } from "@tanstack/react-table";
import { useCallback, useEffect, useState } from "react";

export const baseColumns: ColumnDef<Invitation>[] = [
  {
    accessorKey: "domain_id",
    header: "Domain",
    cell: ({ row }) => {
      return (
        <div className="flex flex-col ">
          {typeof row.original.domain_id === "object" ? (
            <>
              <p>{row.original.domain_id.name || ""}</p>
            </>
          ) : (
            <p>{row.original.domain_id}</p>
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
    cell: ({ row }) => {
      return <Badge variant="outline">{row.getValue("relation")}</Badge>;
    },
  },
  {
    id: "actions",
    header: ({ column }) => (
      <DataTableColumnHeader
        column={column}
        title="Actions"
        className="items-center"
      />
    ),
    cell: ({ row }) => {
      const invitation = row.original;
      return <Actions invitation={invitation} />;
    },
  },
];

export const allColumns: ColumnDef<Invitation>[] = [
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
    accessorKey: "domain_id",
    header: "Domain",
    cell: ({ row }) => {
      return (
        <div className="flex flex-col ">
          {typeof row.original.domain_id === "object" ? (
            <>
              <p>{row.original.domain_id.name || ""}</p>
            </>
          ) : (
            <p>{row.original.domain_id}</p>
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
    cell: ({ row }) => {
      return <Badge variant="outline">{row.getValue("relation")}</Badge>;
    },
  },
  {
    accessorKey: "created_at",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Created At" />
    ),
    cell: ({ row }) => {
      return (
        <DisplayTimeCell
          time={row.getValue("created_at")}
          action={Action.Created}
        />
      );
    },
  },
  {
    id: "actions",
    header: ({ column }) => (
      <DataTableColumnHeader
        column={column}
        title="Actions"
        className="items-center"
      />
    ),
    cell: ({ row }) => {
      const invitation = row.original;
      return <Actions invitation={invitation} />;
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

function Actions({ invitation }: { invitation: Invitation }) {
  return (
    <div className="flex gap-2 sm:gap-8">
      <AcceptInvitationButton
        domainId={
          typeof invitation.domain_id === "string"
            ? invitation.domain_id
            : (invitation.domain_id?.id as string)
        }
      />
      <DeclineInvitationButton
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
}
