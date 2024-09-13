"use client";
import { DomainStatusDialog } from "@/components/entities/domain-status";
import { Icons } from "@/components/icons";
import { RequiredAsterisk } from "@/components/required";
import { Button } from "@/components/ui/button";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { TableCell, TableHead, TableRow } from "@/components/ui/table";
import { UpdateServerSession } from "@/lib/actions";
import { UpdateDomain } from "@/lib/domains";
import { cn } from "@/lib/utils";
import type { Domain, Status } from "@absmach/magistrala-sdk";
import { zodResolver } from "@hookform/resolvers/zod";
import { Pencil } from "lucide-react";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";
import { z } from "zod";
export function UpdateAlias({ alias, id }: { alias: string; id: string }) {
  const [isEditing, setIsEditing] = useState(false);

  const formSchema = z.object({
    // biome-ignore lint/style/useNamingConvention: This is from an external library
    alias: z.string({ required_error: "Alias is required" }),
  });
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      alias: alias,
    },
  });

  async function onSubmit(values: z.infer<typeof formSchema>) {
    const domain: Domain = {
      id: id,
      alias: values.alias,
    };

    const toastId = toast("Sonner");
    toast.loading("Updating domain alias ...", {
      id: toastId,
    });

    const result = await UpdateDomain(domain);
    if (result.error === null) {
      toast.success(`Domain ${result.data} updated successfully`, {
        id: toastId,
      });
      UpdateServerSession();
      setIsEditing(false);
    } else {
      toast.error(`Failed to update domain alias: ${result.error}`, {
        id: toastId,
      });
    }
  }

  return (
    <TableRow>
      <TableHead>Alias {isEditing && <RequiredAsterisk />}</TableHead>

      <TableCell>
        <Form {...form}>
          <form
            onSubmit={form.handleSubmit(onSubmit)}
            className=" flex flex-row justify-between"
          >
            <FormField
              control={form.control}
              name="alias"
              defaultValue={alias}
              render={({ field }) => (
                <FormItem className="w-full">
                  <FormControl className="w-4/5 ">
                    <Input
                      placeholder="Enter alias "
                      {...field}
                      disabled={!isEditing}
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
                      form.reset();
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
      </TableCell>
    </TableRow>
  );
}

export function UpdateStatusDialog({
  status,
  domain,
}: {
  status: Status;
  domain: Domain;
}) {
  const isEntityEnabled = status === "enabled";
  const [open, setOpen] = useState(false);

  return (
    <>
      <Button
        variant="outline"
        className={cn(
          isEntityEnabled
            ? "border-red-500 text-red-500 hover:text-red-700 hover:border-red-700"
            : "border-green-500 text-green-500",
        )}
        onClick={() => setOpen(true)}
      >
        {isEntityEnabled ? "Disable" : "Enable"}
      </Button>
      <DomainStatusDialog
        showStatusDialog={open}
        setShowStatusDialog={setOpen}
        name={domain.name as string}
        id={domain.id as string}
        isEnabled={isEntityEnabled}
      />
    </>
  );
}
