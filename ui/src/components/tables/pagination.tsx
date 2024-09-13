import {
  ChevronLeft,
  ChevronRight,
  ChevronsLeft,
  ChevronsRight,
} from "lucide-react";

import type { Table } from "@tanstack/react-table";
import { Button } from "../ui/button";

import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

import { createPageUrl } from "@/lib/utils";
import clsx from "clsx";
import Link from "next/link";
import { usePathname, useSearchParams } from "next/navigation";

interface DataTablePaginationProps<TableData> {
  table: Table<TableData>;
}

export function DataTablePagination<TableData>({
  table,
}: DataTablePaginationProps<TableData>) {
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const currentPage = Number(searchParams.get("page")) || 1;
  const totalPages = table.getPageCount() || 0;
  const pageSize = table.getState().pagination.pageSize;

  return (
    <div className="flex justify-end items-center space-x-6 lg:space-x-8">
      <div className="flex items-center space-x-2">
        <p className="text-sm font-medium">Rows per page</p>
        <DropdownMenu>
          <DropdownMenuTrigger asChild={true}>
            <Button variant="outline">{pageSize}</Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent>
            {[5, 10, 20, 30, 40, 50].map((pageSize) => (
              <Link
                key={pageSize}
                href={createPageUrl(searchParams, pathname, pageSize, "limit")}
              >
                <DropdownMenuItem>{pageSize}</DropdownMenuItem>
              </Link>
            ))}
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
      <div className="flex w-[100px] items-center justify-center text-sm font-medium">
        Page {totalPages > 0 ? table.getState().pagination.pageIndex + 1 : 0} of{" "}
        {totalPages}
      </div>
      <div className="flex items-center space-x-2">
        <PaginationDoubleArrow
          direction="left"
          href={createPageUrl(searchParams, pathname, 1, "page")}
          isDisabled={currentPage <= 1}
        />
        <PaginationSingleArrow
          direction="left"
          href={createPageUrl(searchParams, pathname, currentPage - 1, "page")}
          isDisabled={currentPage <= 1}
        />
        <PaginationSingleArrow
          direction="right"
          href={createPageUrl(searchParams, pathname, currentPage + 1, "page")}
          isDisabled={currentPage >= totalPages}
        />
        <PaginationDoubleArrow
          direction="right"
          href={createPageUrl(searchParams, pathname, totalPages, "page")}
          isDisabled={currentPage >= totalPages}
        />
      </div>
    </div>
  );
}

function PaginationDoubleArrow({
  href,
  direction,
  isDisabled,
}: {
  href: string;
  direction: "left" | "right";
  isDisabled?: boolean;
}) {
  const className = clsx(
    "flex h-8 w-8 items-center justify-center rounded-md border",
    {
      "pointer-events-none bg-accent text-black midnightsky:text-white":
        isDisabled,
      "bg-primary text-white hover:bg-primary/90": !isDisabled,
      "mr-2 md:mr-2": direction === "left",
      "ml-2 md:ml-2": direction === "right",
    },
  );

  const icon =
    direction === "left" ? (
      <>
        <span className="sr-only">Go to first page</span>
        <ChevronsLeft className="h-4 w-4" />{" "}
      </>
    ) : (
      <>
        <span className="sr-only">Go to last page</span>
        <ChevronsRight className="h-4 w-4" />
      </>
    );

  return isDisabled ? (
    <div className={className}>{icon}</div>
  ) : (
    <Link className={className} href={href}>
      {icon}
    </Link>
  );
}

function PaginationSingleArrow({
  href,
  direction,
  isDisabled,
}: {
  href: string;
  direction: "left" | "right";
  isDisabled?: boolean;
}) {
  const className = clsx(
    "flex h-8 w-8 items-center justify-center rounded-md border",
    {
      "pointer-events-none bg-accent text-black": isDisabled,
      "bg-primary text-white hover:bg-primary/90": !isDisabled,
      "mr-2 md:mr-2": direction === "left",
      "ml-2 md:ml-2": direction === "right",
    },
  );

  const icon =
    direction === "left" ? (
      <>
        <span className="sr-only">Go to previous page</span>
        <ChevronLeft className="h-4 w-4" />
      </>
    ) : (
      <>
        <span className="sr-only">Go to next page</span>
        <ChevronRight className="h-4 w-4" />
      </>
    );

  return isDisabled ? (
    <div className={className}>{icon}</div>
  ) : (
    <Link className={className} href={href}>
      {icon}
    </Link>
  );
}
