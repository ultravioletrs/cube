"use client";

import { type EntityFetchData, FetchData } from "@/lib/actions";
import type { EntityType } from "@/types/entities";
import type { PageMetadata } from "@absmach/magistrala-sdk";
import { useCallback, useEffect, useState } from "react";
import type { ControllerRenderProps } from "react-hook-form";
import { useInView } from "react-intersection-observer";
import { toast } from "sonner";
import MultipleSelector, { type Option } from "../ui/multiple-select";

// biome-ignore lint/suspicious/noExplicitAny: This infiniteSelect component is meant to be used by any dataType.
type InfiniteSelectController = ControllerRenderProps<any>;
export const MultipleSelect = ({
  field,
  entityType,
  initData,
  defaultValues,
  id,
  disabled = false,
  className,
}: {
  field: InfiniteSelectController;
  entityType: EntityType;
  initData: EntityFetchData;
  defaultValues?: Option[];
  id?: string;
  disabled?: boolean;
  className?: string;
}) => {
  const [data, setData] = useState<Option[]>();
  const [total, setTotal] = useState(0);
  const [totalLoaded, setTotalLoaded] = useState(0);
  const [offset, setOffset] = useState(20);
  const [ref, inView] = useInView();
  const [value, setValue] = useState<Option[]>(defaultValues as Option[]);
  const [loadMore, setLoadMore] = useState(totalLoaded < total);
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
    }

    if (newData.data.length > 0) {
      setOffset(next);
      const updatedData = [...initData.data, ...newData.data];
      const dataOptions: Option[] = updatedData.map((item) => ({
        value: item.id as string,
        label: (item.name ? item.name : item.id) as string,
      }));
      setData(dataOptions);
      setTotal(newData.total);
      setTotalLoaded(updatedData.length);
    }
  }, [offset, entityType, id, initData]);

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
      const dataOptions: Option[] = initData.data.map((item) => ({
        value: item.id as string,
        label: (item.name ? item.name : item.id) as string,
      }));
      setData(dataOptions);
      setTotal(initData.total);
      setTotalLoaded(initData.data.length);
    }
  }, [initData]);

  useEffect(() => {
    if (total || totalLoaded) {
      setLoadMore(totalLoaded < total);
    }
  }, [total, totalLoaded]);

  return (
    <MultipleSelector
      placeholder={`Select a ${entityType}`}
      options={data}
      defaultOptions={defaultValues}
      value={value}
      disabled={disabled}
      className={className}
      onChange={(value) => {
        setValue(value);
        const fieldArray: string[] = value.map((item) => item.value);
        field.onChange(fieldArray);
      }}
      emptyIndicator={
        <p className="text-center text-md leading-10 text-gray-600 dark:text-gray-400">
          {`No ${entityType} found`}
        </p>
      }
      loadMore={loadMore}
      loadMoreRef={ref}
    />
  );
};
