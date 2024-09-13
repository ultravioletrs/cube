"use client";
import SearchInput from "@/components/tables/search";
import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { cn } from "@/lib/utils";
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
import { useRouter } from "next/navigation";
import { useCallback, useEffect, useState } from "react";
import { DataTablePagination } from "../tables/pagination";
import { DataTableViewOptions } from "./column-toggle";
import Statusbutton from "./status-button";

const queryClient = new QueryClient();

export default function DataTable<
  TableData extends { id?: string; name?: string },
  TableValue,
>({
  baseColumns,
  allColumns,
  searchPlaceHolder,
  currentPage,
  limit,
  total,
  data,
  href,
  clickable = true,
  userId,
  hasTags,
  canFilter,
  showSearchbar,
  filterByIdentity,
  noContentPlaceHolder,
}: {
  baseColumns: ColumnDef<TableData, TableValue>[];
  allColumns: ColumnDef<TableData, TableValue>[];
  searchPlaceHolder: string;
  currentPage: number;
  limit: number;
  total: number;
  // biome-ignore lint/suspicious/noExplicitAny: Data is a generic type
  data: any;
  noContentPlaceHolder: string;
  href?: string;
  clickable?: boolean;
  userId?: string;
  hasTags?: boolean;
  canFilter?: boolean;
  showSearchbar?: boolean;
  filterByIdentity?: boolean;
}) {
  const [sorting, setSorting] = useState<SortingState>([]);
  const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>([]);
  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>({});
  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: 10,
  });
  const [columns, setColumns] =
    useState<ColumnDef<TableData, TableValue>[]>(allColumns);

  const handleResize = useCallback(() => {
    const isSmallScreen = () => window.innerWidth < 768;
    setColumns(isSmallScreen() ? baseColumns : allColumns);
  }, [allColumns, baseColumns]);

  useEffect(() => {
    if (columns) {
      handleResize();
      window.addEventListener("resize", handleResize);
      return () => window.removeEventListener("resize", handleResize);
    }
  }, [handleResize, columns]);

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

  const router = useRouter();
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
      <div className="flex  items-center py-4 gap-4 sm:gap-8">
        <div className=" w-full">
          <SearchInput
            placeholder={searchPlaceHolder}
            hasTags={hasTags}
            canFilter={canFilter}
            showSearchbar={showSearchbar}
            filterByIdentity={filterByIdentity}
          />
        </div>
        <Statusbutton />
        <DataTableViewOptions table={table} />
      </div>
      <div className="rounded-md border mb-3 bg-card">
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
                  onClick={() => {
                    if (clickable) {
                      router.push(`${href}/${row.original.id ?? ""}`);
                    }
                  }}
                  className={cn(
                    clickable &&
                      "cursor-pointer hover:bg-primary/10 dark:hover:bg-accent",
                  )}
                >
                  {row.getVisibleCells().map((cell) => {
                    return (
                      <TableCell
                        key={cell.id}
                        onClick={(e) => {
                          if (cell.column.id === "actions") {
                            e.stopPropagation();
                          }
                        }}
                      >
                        {cell.column.id === "name" &&
                        userId === row.original.id ? (
                          <div>
                            {`${cell.getValue()} `}
                            <Badge className="rounded-md px-1">You</Badge>
                          </div>
                        ) : cell.column.id === "actions" &&
                          userId === row.original.id ? null : cell.column.id ===
                            "id" && userId === row.original.id ? (
                          <>{row.original.id}</>
                        ) : (
                          flexRender(
                            cell.column.columnDef.cell,
                            cell.getContext(),
                          )
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
                  {noContentPlaceHolder}
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

export function DisplayTags({ tags }: { tags: string[] }) {
  if (!tags) {
    return null;
  }
  return (
    <div className="flex space-x-2">
      <span className="max-w-[500px] truncate font-medium">
        {tags.map((tag, index) => (
          <Badge
            // biome-ignore lint/suspicious/noArrayIndexKey: Tags are not unique
            key={index}
            variant="secondary"
            className="rounded-sm px-1 font-normal text-xs mx-1"
          >
            {tag}
          </Badge>
        ))}
      </span>
    </div>
  );
}
