"use client";
import { DataTableViewOptions } from "@/components/tables/column-toggle";
import { DataTablePagination } from "@/components/tables/pagination";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import {
  type ColumnDef,
  type ColumnFiltersState,
  type PaginationState,
  type SortingState,
  type VisibilityState,
  flexRender,
  getCoreRowModel,
  getFilteredRowModel,
  getSortedRowModel,
  useReactTable,
} from "@tanstack/react-table";
import { useEffect, useState } from "react";

const queryClient = new QueryClient();

export default function DataTable<TableData, TableValue>({
  columns,
  currentPage,
  limit,
  total,
  data,
}: {
  columns: ColumnDef<TableData, TableValue>[];
  currentPage: number;
  limit: number;
  total: number;
  // biome-ignore lint/suspicious/noExplicitAny: Data is a generic type
  data: any;
}) {
  const [sorting, setSorting] = useState<SortingState>([]);
  const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>([]);
  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>({});
  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: 10,
  });

  useEffect(() => {
    const handlePaginationChange = (
      newPageIndex: number,
      newPageSize: number,
    ) => {
      setPagination((prevPagination) => ({
        ...prevPagination,
        pageIndex: newPageIndex - 1,
        pageSize: newPageSize,
      }));
    };
    handlePaginationChange(currentPage, limit);
  }, [currentPage, limit]);

  const table = useReactTable({
    data: data ? data : [],
    columns,
    rowCount: total,
    getCoreRowModel: getCoreRowModel(),
    onSortingChange: setSorting,
    getSortedRowModel: getSortedRowModel(),
    onColumnFiltersChange: setColumnFilters,
    getFilteredRowModel: getFilteredRowModel(),
    onColumnVisibilityChange: setColumnVisibility,
    manualPagination: true,
    state: {
      sorting,
      columnFilters,
      columnVisibility,
      pagination,
    },
  });
  return (
    <QueryClientProvider client={queryClient}>
      <div className="flex  items-center py-4 gap-8">
        <DataTableViewOptions table={table} />
      </div>
      <div className="rounded-md border mb-3 bg-white dark:bg-card">
        <Table>
          <TableHeader>
            {table.getHeaderGroups().map((headerGroup) => (
              <TableRow key={headerGroup.id}>
                {headerGroup.headers.map((header) => {
                  return (
                    <TableHead key={header.id}>
                      {header.isPlaceholder
                        ? null
                        : flexRender(
                            header.column.columnDef.header,
                            header.getContext(),
                          )}
                    </TableHead>
                  );
                })}
              </TableRow>
            ))}
          </TableHeader>
          <TableBody>
            {table.getRowModel().rows?.length ? (
              table.getRowModel().rows.map((row) => (
                <TableRow
                  key={row.id}
                  data-state={row.getIsSelected() && "selected"}
                >
                  {row.getVisibleCells().map((cell) => {
                    return (
                      <TableCell key={cell.id}>
                        {flexRender(
                          cell.column.columnDef.cell,
                          cell.getContext(),
                        )}
                      </TableCell>
                    );
                  })}
                </TableRow>
              ))
            ) : (
              <TableRow>
                <TableCell
                  colSpan={columns.length}
                  className="h-24 text-center text-muted-foreground"
                >
                  No invitations sent yet. Get started by sending an invitation
                  to a user.
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>
      <DataTablePagination table={table} />
    </QueryClientProvider>
  );
}
