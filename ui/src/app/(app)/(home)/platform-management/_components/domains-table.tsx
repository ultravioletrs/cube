"use client";
import { Action, DisplayTimeCell } from "@/components/display-time";
import { DomainStatusDialog } from "@/components/entities/domain-status";
import { DataTableColumnHeader } from "@/components/tables/column-header";
import DataTable from "@/components/tables/table";
import { Switch } from "@/components/ui/switch";
import type { Domain, DomainsPage } from "@absmach/magistrala-sdk";
import type { ColumnDef, Row } from "@tanstack/react-table";
import { useState } from "react";

function SwitchDomainStatus({ row }: { row: Row<Domain> }) {
  const isEntityEnabled = row.getValue("status") === "enabled";
  const name: string = row.getValue("name");
  const id: string = row.original.id as string;
  const [showStatusDialog, setShowStatusDialog] = useState(false);
  return (
    <>
      <Switch
        checked={isEntityEnabled}
        onCheckedChange={() => {
          setShowStatusDialog(true);
        }}
      />
      <DomainStatusDialog
        showStatusDialog={showStatusDialog}
        setShowStatusDialog={setShowStatusDialog}
        isEnabled={isEntityEnabled}
        name={name}
        id={id}
      />
    </>
  );
}

export const baseColumns: ColumnDef<Domain>[] = [
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
    cell: SwitchDomainStatus,
  },
];

export const allColumns: ColumnDef<Domain>[] = [
  {
    accessorKey: "name",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Name" />
    ),
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
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Alias" />
    ),
    cell: ({ row }) => {
      return (
        <div className="max-w-[10rem]">
          <span className="break-words">{row.getValue("alias")}</span>
        </div>
      );
    },
  },
  {
    accessorKey: "createdBy",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Created By" />
    ),
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
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Created At" />
    ),
    cell: ({ row }) => {
      return <DisplayTimeCell time={row.getValue("created_at")} />;
    },
  },
  {
    accessorKey: "updatedBy",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Updated By" />
    ),
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
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Updated At" />
    ),
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
    accessorKey: "status",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Status" />
    ),
    cell: SwitchDomainStatus,
  },
];

export function DomainsTable({
  domainsPage,
  limit,
}: {
  domainsPage: DomainsPage;
  limit: number;
}) {
  return (
    <DataTable
      baseColumns={baseColumns}
      allColumns={allColumns}
      searchPlaceHolder="search domains by name"
      currentPage={Math.ceil(domainsPage.offset / domainsPage.limit) + 1}
      total={domainsPage.total}
      limit={limit}
      data={domainsPage.domains}
      clickable={false}
      noContentPlaceHolder="No domains found. Get started by creating one."
    />
  );
}
