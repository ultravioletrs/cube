"use client";
import { ChevronDown, Search as MagnifyingGlassIcon } from "lucide-react";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { useDebouncedCallback } from "use-debounce";
import { Badge } from "../ui/badge";
import { Button } from "../ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "../ui/dropdown-menu";

export default function SearchInput({
  placeholder,
  canFilter = true,
  hasTags = false,
  showSearchbar = true,
  filterByIdentity = false,
}: {
  placeholder: string;
  canFilter?: boolean;
  hasTags?: boolean;
  showSearchbar?: boolean;
  filterByIdentity?: boolean;
}) {
  const searchParams = useSearchParams();
  const pathname = usePathname();
  const { replace } = useRouter();
  const [searchField, setSearchField] = useState("name");
  const inputRef = useRef<HTMLInputElement>(null);

  const handleSearch = useDebouncedCallback((term) => {
    const params = new URLSearchParams(searchParams.toString());
    const searchFields = ["name", "tag", "id", "identity"];

    for (const field of searchFields) {
      if (field !== searchField) {
        params.delete(field);
      }
    }

    if (term) {
      params.set(searchField, term);
    } else {
      params.delete(searchField);
    }

    replace(`${pathname}?${params.toString()}`);
  }, 300);

  useEffect(() => {
    const params = new URLSearchParams(searchParams.toString());
    const searchFields = ["name", "tag", "id", "identity"];

    for (const field of searchFields) {
      if (field !== searchField) {
        params.delete(field);
      }
    }

    replace(`${pathname}?${params.toString()}`);
  }, [searchField, pathname, replace, searchParams]);

  const handleChangeSearchField = (newSearchField: string) => {
    if (newSearchField !== searchField) {
      setSearchField(newSearchField);
      if (inputRef.current) {
        inputRef.current.value = "";
      }
    }
  };

  return (
    <>
      {showSearchbar && (
        <div className="relative">
          <label htmlFor="search" className="sr-only">
            Search by {searchField}
          </label>
          <input
            ref={inputRef}
            className="peer block w-full rounded-md border bg-card border-gray-200 py-[9px] pl-10 text-sm outline-2 placeholder:text-gray-500"
            placeholder={placeholder}
            onChange={(e) => {
              handleSearch(e.target.value);
            }}
          />
          <MagnifyingGlassIcon className="absolute left-3 top-1/2 h-[18px] w-[18px] -translate-y-1/2 text-gray-500 peer-focus:text-gray-900" />
          {canFilter && (
            <DropdownMenu>
              <DropdownMenuTrigger asChild={true}>
                <Button
                  variant="outline"
                  className="absolute right-0 top-1/2 bg-transparent border-0 hover:bg-transparent -translate-y-1/2 pr-1"
                >
                  {searchField && (
                    <div className="text-center flex flex-row ">
                      <Badge
                        variant="outline"
                        className="ml-2 rounded text-card-foreground"
                      >
                        {searchField}
                      </Badge>
                    </div>
                  )}
                  <ChevronDown className="ml-2 w-4 h-4 text-muted-foreground" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuLabel>Search by:</DropdownMenuLabel>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  onClick={() => handleChangeSearchField("name")}
                  className={searchField === "name" ? "bg-primary/10" : ""}
                >
                  Name
                </DropdownMenuItem>
                {hasTags && (
                  <DropdownMenuItem
                    onClick={() => handleChangeSearchField("tag")}
                    className={searchField === "tag" ? "bg-primary/10" : ""}
                  >
                    Tag
                  </DropdownMenuItem>
                )}
                <DropdownMenuItem
                  onClick={() => handleChangeSearchField("id")}
                  className={searchField === "id" ? "bg-primary/10" : ""}
                >
                  ID
                </DropdownMenuItem>
                {filterByIdentity && (
                  <DropdownMenuItem
                    onClick={() => handleChangeSearchField("identity")}
                    className={searchField === "id" ? "bg-primary/10" : ""}
                  >
                    Identity
                  </DropdownMenuItem>
                )}
              </DropdownMenuContent>
            </DropdownMenu>
          )}
        </div>
      )}
    </>
  );
}
