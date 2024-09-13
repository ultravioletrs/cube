import { SearchUsers } from "@/lib/users";
import { useState } from "react";
import type { ControllerRenderProps } from "react-hook-form";
import MultipleSelector, { type Option } from "../ui/multiple-select";

// biome-ignore lint/suspicious/noExplicitAny: This infiniteSelect component is meant to be used by any dataType.
type InfiniteSelectController = ControllerRenderProps<any>;
interface UserSearchInputProps {
  field: InfiniteSelectController;
  defaultValues?: Option[];
}

export default function UserSearchInput({
  field,
  defaultValues,
}: UserSearchInputProps) {
  const [searchResults, setSearchResults] = useState<Option[]>([]);
  const [value, setValue] = useState<Option[]>(defaultValues as Option[]);
  const handleSearch = async (value: string): Promise<Option[]> => {
    if (value.trim() === "") {
      setSearchResults([]);
      return Promise.resolve([]);
    }

    const response = await SearchUsers({ name: value });
    if (response.error !== null) {
      setSearchResults([]);
      return Promise.resolve([]);
    }

    const dataOptions: Option[] = response.data?.users.map((user) => ({
      value: user.id as string,
      label: user.name as string,
    })) as Option[];
    setSearchResults(dataOptions);
    return dataOptions;
  };

  return (
    <MultipleSelector
      placeholder="Select user(s)"
      options={searchResults}
      emptyIndicator={
        <p className="text-center text-md leading-10 text-gray-600 dark:text-gray-400">
          {"No user found"}
        </p>
      }
      onSearch={(value) => {
        return handleSearch(value);
      }}
      defaultOptions={defaultValues}
      value={value}
      onChange={(value) => {
        setValue(value);
        const fieldArray: string[] = value.map((item) => item.value);
        field.onChange(fieldArray);
      }}
    />
  );
}
