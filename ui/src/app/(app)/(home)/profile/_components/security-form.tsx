"use client";
import { PasswordFormInput } from "@/components/entities/password";
import { RequiredAsterisk } from "@/components/required";
import { Button } from "@/components/ui/button";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { UpdateUserPassword } from "@/lib/users";
import { zodResolver } from "@hookform/resolvers/zod";
import { signOut } from "next-auth/react";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { Toaster, toast } from "sonner";
import { z } from "zod";

const formSchema = z
  .object({
    currentPassword: z.string().min(8, {
      message: "Password must be 8 characters long",
    }),
    newPassword: z.string().min(8, {
      message: "Password must be 8 characters long",
    }),
    confirmPassword: z.string().min(8, {
      message: "Password must be 8 characters long",
    }),
  })
  .refine((data) => data.newPassword === data.confirmPassword, {
    message: "Passwords do not match",
    path: ["confirmPassword"],
  });

export default function SecurityForm() {
  const [processing, setProcessing] = useState(false);
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
  });

  const loginUrl = `${process.env.NEXT_PUBLIC_BASE_URL}/login`;

  async function onSubmit(values: z.infer<typeof formSchema>) {
    setProcessing(true);
    const oldSecret = values.currentPassword;
    const newSecret = values.newPassword;

    const toastId = toast("Sonner");

    toast.loading("Updating Account Details...", {
      id: toastId,
    });

    const result = await UpdateUserPassword(oldSecret, newSecret);
    if (result.error) {
      toast.error("Failed to Update Account Details.", {
        id: toastId,
      });
      setProcessing(false);
      form.reset();
      return;
    }

    toast.success("Account Details updated successfully.", {
      id: toastId,
    });
    signOut({ callbackUrl: loginUrl });
    setProcessing(false);
    form.reset();
  }

  return (
    <Form {...form}>
      <form
        onSubmit={form.handleSubmit(onSubmit)}
        className="space-y-4 md:space-y-8"
      >
        <FormField
          control={form.control}
          name="currentPassword"
          render={({ field }) => (
            <FormItem>
              <FormLabel>
                Current Password <RequiredAsterisk />
              </FormLabel>
              <FormControl>
                <PasswordFormInput processing={processing} field={field} />
              </FormControl>
              <FormMessage {...field} />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name="newPassword"
          render={({ field }) => (
            <FormItem>
              <FormLabel>
                New Password <RequiredAsterisk />
              </FormLabel>
              <FormControl>
                <PasswordFormInput processing={processing} field={field} />
              </FormControl>
              <FormMessage {...field} />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name="confirmPassword"
          render={({ field }) => (
            <FormItem>
              <FormLabel>
                Confirm Password <RequiredAsterisk />
              </FormLabel>
              <FormControl>
                <PasswordFormInput processing={processing} field={field} />
              </FormControl>
              <FormMessage {...field} />
            </FormItem>
          )}
        />
        <Toaster richColors={true} expand={true} visibleToasts={1} />
        <Button type="submit" disabled={processing}>
          Update
        </Button>
      </form>
    </Form>
  );
}
