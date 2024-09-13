"use client";
import CodeMirrorEditor from "@/components/codemirror";
import { PasswordFormInput } from "@/components/entities/password";
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
import { CreateUser } from "@/lib/users";
import type { Metadata } from "@/types/entities";
import type { User } from "@absmach/magistrala-sdk";
import { zodResolver } from "@hookform/resolvers/zod";
import type { Tag } from "emblor";
import { Plus } from "lucide-react";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";
import { z } from "zod";

const formSchema = z.object({
  // biome-ignore lint/style/useNamingConvention: This is from an external library
  name: z.string({ required_error: "Name is required" }),
  email: z.string().email({ message: "Invalid email address" }),
  password: z.string().min(8, {
    message: "Password must be atleast 8 characters.",
  }),
  tags: z.array(z.object({ id: z.string(), text: z.string() })).optional(),
  metadata: z.string().optional(),
});

export const CreateUserForm = () => {
  const [processing, setProcessing] = useState(false);
  const [open, setOpen] = useState(false);
  const [newTags, setNewTags] = useState<Tag[]>(StringArrayToTags([]));

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
  });

  async function onSubmit(values: z.infer<typeof formSchema>) {
    setProcessing(true);
    let metadata: Metadata = {};
    if (values.metadata) {
      metadata = JSON.parse(values.metadata as string);
    }
    const user: User = {
      name: values.name,
      credentials: {
        identity: values.email,
        secret: values.password,
      },
      tags: TagsToStringArray(newTags),
      metadata: { admin: metadata },
    };

    const toastId = toast("Sonner");
    toast.loading("Creating user...", {
      id: toastId,
    });

    const result = await CreateUser(user);

    setProcessing(false);

    if (result.error === null) {
      toast.success(`User ${result.data} created successfully`, {
        id: toastId,
      });
      form.reset();
      setNewTags(StringArrayToTags([]));
      setOpen(false);
    } else {
      toast.error(`Failed to create user: ${result.error}`, {
        id: toastId,
      });
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild={true}>
        <Button>
          <Plus className="h-5 md:mr-2" />
          <span>Create</span>
        </Button>
      </DialogTrigger>
      <DialogContent className="rounded max-w-[400px] sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle>Create User</DialogTitle>
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
                    <Input placeholder="Enter name " {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="email"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    Email address <RequiredAsterisk />
                  </FormLabel>
                  <FormControl>
                    <Input placeholder="Enter email " {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="password"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    Password <RequiredAsterisk />
                  </FormLabel>
                  <FormControl>
                    <PasswordFormInput processing={processing} field={field} />
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
                <Button
                  type="button"
                  variant="secondary"
                  onClick={() => {
                    form.reset();
                    setNewTags(StringArrayToTags([]));
                  }}
                >
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
