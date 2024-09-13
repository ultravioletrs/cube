import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Spinner } from "@/components/ui/spinner";
import { type EntityFetchData, FetchData } from "@/lib/actions";
import type { EntityType } from "@/types/entities";
import type { PageMetadata, User } from "@absmach/magistrala-sdk";
import { useCallback, useEffect, useRef, useState } from "react";
import type { ControllerRenderProps } from "react-hook-form";
import { useInView } from "react-intersection-observer";
import { toast } from "sonner";
import { useDebouncedCallback } from "use-debounce";
import { Button } from "../ui/button";
import { FormControl, FormMessage } from "../ui/form";
import { Input } from "../ui/input";

// biome-ignore lint/suspicious/noExplicitAny: This infiniteSelect component is meant to be used by any dataType.
type InfiniteSelectController = ControllerRenderProps<any>;

export const InfiniteSelect = ({
  field,
  entityType,
  initData,
  id,
  onChange,
  className,
}: {
  field: InfiniteSelectController;
  entityType: EntityType;
  initData: EntityFetchData;
  id?: string;
  onChange?: (value: string | undefined) => void;
  className?: string;
}) => {
  const [data, setData] = useState<User[]>([]);
  const [total, setTotal] = useState(0);
  const [totalLoaded, setTotalLoaded] = useState(0);
  const [offset, setOffset] = useState(20);
  const [ref, inView] = useInView();
  const [searchTerm, setSearchTerm] = useState("");
  const [selectedValue, setSelectedValue] = useState<string | undefined>(
    field.value,
  );
  const [key, setKey] = useState(+new Date());
  const inputRef = useRef<HTMLInputElement>(null);
  const [error, setError] = useState<string | null>(initData?.error);
  const limit: number = 20;
  const loadMoreEntities = useCallback(async () => {
    const next = offset + limit;
    const queryParams: PageMetadata = {
      offset: offset,
      limit: 20,
    };
    const newData = await FetchData(entityType, queryParams, id);

    if (newData.error) {
      setError(newData.error);
      toast.error(`Error fetching data: ${newData.error}`);
      return;
    }

    if (newData.data.length > 0) {
      setOffset(next);
      const updatedData = [...data, ...newData.data];
      setData(updatedData);
      setTotal(newData.total);
      setTotalLoaded(updatedData.length);
    }
  }, [offset, data, entityType, id]);

  const fetchData = useCallback(
    async (searchTerm: string) => {
      const queryParams: PageMetadata = {
        offset: 0,
        limit: 20,
        name: searchTerm,
      };
      const newData = await FetchData(entityType, queryParams, id);

      if (newData.error) {
        setError(newData.error);
        toast.error(`Error fetching data: ${newData.error}`);
      } else {
        setOffset(20);
      }

      setData(newData.data);
      setTotal(newData.total);
      setTotalLoaded(newData.data.length);
      setOffset(20);
    },
    [entityType, id],
  );

  useEffect(() => {
    if (inView && !error) {
      (async () => {
        await new Promise((resolve) => setTimeout(resolve, 100));
        await loadMoreEntities();
      })();
    }
  }, [inView, loadMoreEntities, error]);

  useEffect(() => {
    if (initData?.error) {
      setError(initData.error);
      toast.error(`Error fetching data: ${initData.error}`);
    }

    if (initData) {
      setData(initData.data);
      setTotal(initData.total);
      setTotalLoaded(initData.data.length);
    }
  }, [initData]);

  const handleSearch = useDebouncedCallback((term) => {
    if (inputRef.current) {
      inputRef.current.focus();
    }
    if (term) {
      setSearchTerm(term);
      fetchData(term);
    } else {
      fetchData("");
    }
  }, 500);

  return (
    <>
      <Select
        onValueChange={(value) => {
          setSelectedValue(value);
          field.onChange(value);
          if (onChange) {
            onChange(value);
          }
        }}
        defaultValue={field.value}
        value={selectedValue}
      >
        <FormControl>
          <SelectTrigger className="min-w-[10rem]">
            <SelectValue
              placeholder={
                searchTerm && !selectedValue
                  ? `No ${entityType} found`
                  : `Select a ${entityType}`
              }
            />
          </SelectTrigger>
        </FormControl>
        <SelectContent className={className}>
          <Input
            className="whitespace-nowrap overflow-hidden text-ellipsis"
            ref={inputRef}
            key={key}
            defaultValue={searchTerm}
            onChange={(e) => {
              e.stopPropagation();
              handleSearch(e.target.value);
            }}
            placeholder={`Search ${entityType} by name`}
          />
          <Button
            className="my-2 w-full"
            variant="ghost"
            size="sm"
            onClick={(e) => {
              e.stopPropagation();
              setSelectedValue("");
              field.onChange("");
              setSearchTerm("");
              fetchData("");
              setKey(+new Date());
              if (onChange) {
                onChange(undefined);
              }
            }}
          >
            Clear Selection
          </Button>
          <SelectGroup className="overflow-y-auto max-h-[15rem]">
            {data?.length && data.length > 0 ? (
              data.map((entity) => (
                <SelectItem
                  key={entity.id}
                  value={entity.id as string}
                  className="break-all"
                >
                  {entity.name ? entity.name : entity.id}
                </SelectItem>
              ))
            ) : (
              <SelectLabel>{`No ${entityType} found`}</SelectLabel>
            )}
            {totalLoaded < total ? (
              <div ref={ref}>
                <Spinner />
              </div>
            ) : null}
          </SelectGroup>
        </SelectContent>
      </Select>
      <FormMessage />
    </>
  );
};
