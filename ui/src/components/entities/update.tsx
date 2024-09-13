"use client";

import { Icons } from "@/components/icons";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { UpdateServerSession } from "@/lib/actions";
import { UpdateMetadata, UpdateName, UpdateTags } from "@/lib/entities";
import { toSentenseCase } from "@/lib/utils";
import { EntityType, type Metadata } from "@/types/entities";
import { zodResolver } from "@hookform/resolvers/zod";
import { type Tag, TagInput } from "emblor";
import { Pencil } from "lucide-react";
import React, { type Dispatch, type SetStateAction, useState } from "react";
import { type ControllerRenderProps, useForm } from "react-hook-form";
import { toast } from "sonner";
import { z } from "zod";
import CodeMirrorEditor from "../codemirror";
import { RequiredAsterisk } from "../required";
import { TableCell, TableHead, TableRow } from "../ui/table";

type UpdateMetadataProps = {
  metadata: string;
  id: string;
  entity: EntityType;
};

export default function UpdateMetadataDialog({
  metadata,
  id,
  entity,
}: UpdateMetadataProps) {
  if (metadata === undefined) {
    metadata = "{}";
  }

  const parsedMetadata = JSON.parse(metadata);
  const adminMetadata = parsedMetadata.admin || {};

  const [open, setOpen] = useState(false);
  const [value, setValue] = useState(
    JSON.stringify(
      entity === EntityType.User ? adminMetadata : parsedMetadata,
      null,
      2,
    ),
  );

  async function onSubmit() {
    let updatedMetadata: Metadata = {};
    if (value) {
      if (entity === EntityType.User) {
        updatedMetadata = {
          ...parsedMetadata,
          admin: JSON.parse(value as string),
        };
      } else {
        updatedMetadata = JSON.parse(value as string);
      }
    }

    const toastId = toast("Sonner");
    toast.loading(`Updating ${entity} ...`, {
      id: toastId,
    });

    await UpdateMetadata(id, entity, updatedMetadata);
    setOpen(false);
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild={true}>
        <Button variant="outline" size="icon" className="hover:bg-primary/10">
          <Pencil className="h-4 w-4" />
        </Button>
      </DialogTrigger>
      <DialogContent className="overflow-y-auto md:max-h-[700px] rounded max-w-[400px] md:max-w-[800px]">
        <DialogHeader>
          <DialogTitle>Update {toSentenseCase(entity)} Metadata</DialogTitle>
        </DialogHeader>
        <CodeMirrorEditor value={value} onChange={setValue} height="500px" />
        <DialogFooter className="flex flex-row justify-end gap-2">
          <DialogClose asChild={true}>
            <Button
              type="button"
              variant="secondary"
              onClick={() => {
                setOpen(false);
                setValue(
                  JSON.stringify(
                    entity === EntityType.User ? adminMetadata : parsedMetadata,
                    null,
                    2,
                  ),
                );
              }}
            >
              Close
            </Button>
          </DialogClose>
          <Button
            type="submit"
            onClick={async () => {
              await onSubmit();
              document.location.reload();
            }}
          >
            Update
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export function StringArrayToTags(tags: string[]): Tag[] {
  const tagsArray: Tag[] = [];
  tags.map((tag) => tagsArray.push({ id: tag, text: tag }));
  return tagsArray;
}

export function TagsToStringArray(tags: Tag[]): string[] {
  const updatedTags: string[] = [];
  for (const tag of tags) {
    updatedTags.push(tag.text);
  }
  return updatedTags;
}

type UpdateTagsProps = {
  tags: string[];
  id: string;
  entity: EntityType;
};

export function UpdateTagsDialog({ tags, id, entity }: UpdateTagsProps) {
  const [open, setOpen] = useState(false);
  const [newTags, setNewTags] = useState<Tag[]>(StringArrayToTags(tags));

  const formSchema = z.object({
    tags: z.array(z.object({ id: z.string(), text: z.string() })).optional(),
  });

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      tags: [],
    },
  });

  async function onSubmit(_values: z.infer<typeof formSchema>) {
    const toastId = toast("Sonner");
    toast.loading(`Updating ${entity} ...`, {
      id: toastId,
    });

    const result = await UpdateTags(id, TagsToStringArray(newTags), entity);
    if (result.error === null) {
      toast.success(
        `${toSentenseCase(entity)} ${result.data} updated successfully`,
        {
          id: toastId,
        },
      );
      setOpen(false);
    } else {
      toast.error(`Failed to update ${entity}: ${result.error}`, {
        id: toastId,
      });
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild={true}>
        <Button variant="outline" size="icon" className="hover:bg-primary/10">
          <Pencil className="h-4 w-4" />
        </Button>
      </DialogTrigger>
      <DialogContent className="rounded max-w-[400px] sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle>Update {toSentenseCase(entity)}</DialogTitle>
        </DialogHeader>
        <Form {...form}>
          <form
            onSubmit={form.handleSubmit(onSubmit)}
            className="space-y-4 md:space-y-8 flex flex-col"
          >
            <FormField
              control={form.control}
              name="tags"
              render={({ field }) => (
                <TagsFormInput
                  newTags={newTags}
                  setTags={setNewTags}
                  field={field}
                />
              )}
            />
            <DialogFooter className="flex flex-row justify-end gap-2">
              <DialogClose asChild={true}>
                <Button
                  type="button"
                  variant="secondary"
                  onClick={() => form.reset()}
                >
                  Close
                </Button>
              </DialogClose>
              <Button type="submit">Update</Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
}

type UpdateTagsController = ControllerRenderProps<
  {
    tags?: { id: string; text: string }[] | undefined;
  },
  "tags"
>;

type CreateThingController = ControllerRenderProps<
  {
    name: string;
    key?: string | undefined;
    tags?: { id: string; text: string }[] | undefined;
    metadata?: string | undefined;
  },
  "tags"
>;

type CreateUserController = ControllerRenderProps<
  {
    name: string;
    email: string;
    password: string;
    tags?: { id: string; text: string }[] | undefined;
    metadata?: string | undefined;
  },
  "tags"
>;

type CreateDomainController = ControllerRenderProps<
  {
    name: string;
    alias?: string;
    tags?: { id: string; text: string }[] | undefined;
    metadata?: string | undefined;
  },
  "tags"
>;

type CreateDashboardController = ControllerRenderProps<
  {
    name: string;
    description?: string | undefined;
    tags?: string[] | undefined;
  },
  "tags"
>;

export function TagsFormInput({
  newTags,
  setTags,
  field,
}: {
  newTags: Tag[];
  setTags: Dispatch<SetStateAction<Tag[]>>;
  field:
    | UpdateTagsController
    | CreateThingController
    | CreateUserController
    | CreateDomainController
    | CreateDashboardController;
}) {
  const [activeTagIndex, setActiveTagIndex] = React.useState<number | null>(
    null,
  );
  return (
    <FormItem className="flex flex-col items-start">
      <FormLabel className="text-left">Tags</FormLabel>
      <FormControl className="w-full">
        <TagInput
          {...field}
          placeholder="Enter tags"
          tags={newTags}
          setTags={(updatedTags) => {
            setTags(updatedTags);
          }}
          showCounter={true}
          placeHolder="Enter tags"
          name="tags"
          shape="rounded"
          truncate={45}
          className="pl-2"
          activeTagIndex={activeTagIndex}
          setActiveTagIndex={setActiveTagIndex}
        />
      </FormControl>
      <FormMessage />
    </FormItem>
  );
}

type UpdateNameProps = {
  name: string;
  id: string;
  entity: EntityType;
  canEdit?: boolean;
};

export function UpdateNameDialog({
  name,
  id,
  entity,
  canEdit = true,
}: UpdateNameProps) {
  const [isEditing, setIsEditing] = useState(false);
  const formSchema = z.object({
    // biome-ignore lint/style/useNamingConvention: This is from an external library
    name: z.string({ required_error: "Name is required" }),
  });
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      name: name,
    },
  });

  async function onSubmit(values: z.infer<typeof formSchema>) {
    const toastId = toast("Sonner");
    toast.loading(`Updating ${entity} ...`, {
      id: toastId,
    });

    const result = await UpdateName(id, values.name, entity);
    if (result.error === null) {
      toast.success(
        `${toSentenseCase(entity)} ${result.data} updated successfully`,
        {
          id: toastId,
        },
      );
      if (entity === EntityType.Domain) {
        UpdateServerSession();
      }
      setIsEditing(false);
    } else {
      toast.error(`Failed to update ${entity}: ${result.error}`, {
        id: toastId,
      });
    }
  }

  return (
    <TableRow>
      <TableHead>Name {isEditing && <RequiredAsterisk />}</TableHead>

      <TableCell>
        {canEdit ? (
          <Form {...form}>
            <form
              onSubmit={form.handleSubmit(onSubmit)}
              className=" flex flex-row justify-between"
            >
              <FormField
                control={form.control}
                name="name"
                defaultValue={name}
                render={({ field }) => (
                  <FormItem className="w-full ">
                    <FormControl className="w-4/5 ">
                      <Input
                        {...field}
                        disabled={!isEditing}
                        className="truncate ..."
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <div className="flex flex-row gap-2">
                {isEditing ? (
                  <div className="flex flex-row gap-2">
                    <Button type="submit" variant="outline" size="icon">
                      <Icons.approve className="h-5 w-5" />
                    </Button>
                    <Button
                      type="button"
                      variant="outline"
                      size="icon"
                      onClick={() => {
                        form.reset({ name: name }, { keepValues: false });
                        setIsEditing(false);
                      }}
                    >
                      <Icons.discard className="h-5 w-5" />
                    </Button>
                  </div>
                ) : (
                  <Button
                    type="button"
                    variant="outline"
                    size="icon"
                    onClick={() => setIsEditing(true)}
                    className="hover:bg-primary/10"
                  >
                    <Pencil className="h-4 w-4" />
                  </Button>
                )}
              </div>
            </form>
          </Form>
        ) : (
          <div>{name}</div>
        )}
      </TableCell>
    </TableRow>
  );
}
