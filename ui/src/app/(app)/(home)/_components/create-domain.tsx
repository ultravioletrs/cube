"use client";
import CodeMirrorEditor from "@/components/codemirror";
import {
  StringArrayToTags,
  TagsFormInput,
  TagsToStringArray,
} from "@/components/entities/update";
import { RequiredAsterisk } from "@/components/required";
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
import { CreateDomain } from "@/lib/domains";
import type { Metadata } from "@/types/entities";
import type { Domain } from "@absmach/magistrala-sdk";
import { zodResolver } from "@hookform/resolvers/zod";
import type { Tag } from "emblor";
import { Plus } from "lucide-react";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";
import { z } from "zod";

const formSchema = z.object({
  name: z.string({
    // biome-ignore lint/style/useNamingConvention: This is from an external library
    required_error: "Name is required",
  }),
  alias: z.string({
    // biome-ignore lint/style/useNamingConvention: This is from an external library
    required_error: "Alias is required",
  }),
  tags: z.array(z.object({ id: z.string(), text: z.string() })).optional(),
  metadata: z.string().optional(),
});

export const CreateDomainForm = () => {
  const [open, setOpen] = useState(false);
  const [newTags, setNewTags] = useState<Tag[]>(StringArrayToTags([]));

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      tags: [],
      metadata: "{}",
    },
  });

  async function onSubmit(values: z.infer<typeof formSchema>) {
    let metadata: Metadata = {};
    if (values.metadata) {
      metadata = JSON.parse(values.metadata as string);
    }

    const domain: Domain = {
      name: values.name,
      alias: values.alias,
      tags: TagsToStringArray(newTags),
      metadata: metadata,
    };

    const toastId = toast("Sonner");
    toast.loading("Creating domain...", {
      id: toastId,
    });

    const result = await CreateDomain(domain);
    if (result.error === null) {
      toast.success(`Domain ${result.data} created successfully`, {
        id: toastId,
      });
      form.reset();
      setOpen(false);
    } else {
      toast.error(`Failed to create domain: ${result.error}`, {
        id: toastId,
      });
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild={true}>
        <Button>
          <Plus className="h-5 mr-2" />
          <span>Create</span>
        </Button>
      </DialogTrigger>
      <DialogContent className="rounded max-w-[400px] sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle>Create Domain</DialogTitle>
        </DialogHeader>
        <Form {...form}>
          <form
            onSubmit={form.handleSubmit(onSubmit)}
            className="space-y-4 md:space-y-8"
          >
            <FormField
              control={form.control}
              name="name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    Name <RequiredAsterisk />
                  </FormLabel>
                  <FormControl>
                    <Input placeholder="Enter domain name " {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="alias"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    Alias <RequiredAsterisk />
                  </FormLabel>
                  <FormControl>
                    <Input placeholder="Enter domain alias" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
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
            <FormField
              control={form.control}
              name="metadata"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Metadata</FormLabel>
                  <FormControl>
                    <CodeMirrorEditor
                      value={field.value || "{}"}
                      onChange={field.onChange}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <DialogFooter className="flex flex-row justify-end gap-2">
              <DialogClose asChild={true}>
                <Button type="button" variant="secondary">
                  Close
                </Button>
              </DialogClose>
              <Button type="submit">Create</Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
};
