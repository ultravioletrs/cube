"use client";
import { Action, DisplayTimeCell } from "@/components/display-time";
import { DisplayStatusWithIcon } from "@/components/entities/status-display-with-icon";
import { DataTableColumnHeader } from "@/components/tables/column-header";
import DataTable from "@/components/tables/table";
import { DomainLogin } from "@/lib/actions";
import type { Domain, DomainsPage } from "@absmach/magistrala-sdk";
import type { ColumnDef, Row } from "@tanstack/react-table";
import { LogIn } from "lucide-react";

const SwitchDomainCellComponent = ({ row }: { row: Row<Domain> }) => {
  return (
    <button
      type="button"
      className="justify-self-center p-2 rounded-md text-sm font-medium hover:bg-accent hover:text-accent-foreground"
      onClick={async () => {
        await DomainLogin(row.original.id as string);
      }}
    >
      <LogIn className="h-6 w-6  " />
    </button>
  );
};

export const baseColumns: ColumnDef<Domain>[] = [
  {
    accessorKey: "name",
    header: "Name",
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
    id: "switch",
    cell: SwitchDomainCellComponent,
  },
];

export const allColumns: ColumnDef<Domain>[] = [
  {
    accessorKey: "name",
    header: "Name",
    cell: ({ row }) => {
      return (
        <div className="max-w-[10rem] ">
          <span className="break-words">{row.getValue("name")}</span>
        </div>
      );
    },
  },
  {
    accessorKey: "alias",
    header: "Alias",
    cell: ({ row }) => {
      return (
        <div className="max-w-[10rem]">
          <span className="break-words">{row.getValue("alias")}</span>
        </div>
      );
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
    accessorKey: "permission",
    header: "Permission",
  },
  {
    accessorKey: "createdBy",
    header: "Created By",
    cell: ({ row }) => {
      return (
        <div className="flex flex-col">
          {typeof row.original.created_by === "object" ? (
            <>
              <div>{row.original.created_by.name || ""}</div>
            </>
          ) : (
            <div>{row.original.created_by}</div>
          )}
        </div>
      );
    },
  },
  {
    accessorKey: "created_at",
    header: "Created At",
    cell: ({ row }) => {
      return <DisplayTimeCell time={row.getValue("created_at")} />;
    },
  },
  {
    accessorKey: "updatedBy",
    header: "Updated By",
    cell: ({ row }) => {
      return (
        <div className="flex flex-col justify-center align-middle items-center ">
          {typeof row.original.updated_by === "object" ? (
            <>
              <div>{row.original.updated_by.name || ""}</div>
            </>
          ) : (
            <div>{row.original.updated_by}</div>
          )}
        </div>
      );
    },
  },
  {
    accessorKey: "updated_at",
    header: "Updated At",
    cell: ({ row }) => {
      return (
        <DisplayTimeCell
          time={row.getValue("updated_at")}
          action={Action.Updated}
        />
      );
    },
  },

  {
    id: "switch",
    cell: SwitchDomainCellComponent,
  },
];

export function DomainsTable({
  domainsPage,
  limit,
}: {
  domainsPage: DomainsPage;
  limit: number;
}) {
  const filteredDomains: Domain[] = [];
  domainsPage.domains.map((domain) => {
    if (
      (domain.permission !== "administrator" && domain.status === "enabled") ||
      domain.permission === "administrator"
    ) {
      filteredDomains.push(domain);
    }
  });

  return (
    <DataTable
      baseColumns={baseColumns}
      allColumns={allColumns}
      searchPlaceHolder="Search Domains"
      currentPage={Math.ceil(domainsPage.offset / domainsPage.limit) + 1}
      total={domainsPage.total}
      limit={limit}
      data={filteredDomains}
      clickable={false}
      canFilter={false}
      noContentPlaceHolder="No domains found. Get started by creating one."
    />
  );
}
