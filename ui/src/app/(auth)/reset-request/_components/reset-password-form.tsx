"use client";

import { PasswordFormInput } from "@/components/entities/password";
import { Button } from "@/components/ui/button";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { ResetPassword } from "@/lib/users";
import { zodResolver } from "@hookform/resolvers/zod";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";
import { z } from "zod";

const formSchema = z
  .object({
    password: z.string().min(8, {
      message: "Password must be atleast 8 characters.",
    }),
    confirmPassword: z.string(),
  })
  .refine((data) => data.password === data.confirmPassword, {
    message: "Passwords do not match",
    path: ["confirmPassword"],
  });

const ResetPasswordForm = ({
  searchParams,
}: {
  searchParams: { token: string };
}) => {
  const [processing, setProcessing] = useState(false);
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
  });

  async function onsubmit(values: z.infer<typeof formSchema>) {
    setProcessing(true);

    const toastId = toast("sonner");
    toast.loading("Resetting password...", {
      id: toastId,
    });

    const response = await ResetPassword(
      values.password,
      values.confirmPassword,
      searchParams.token,
    );

    if (response?.error) {
      toast.error("Failed to reset password", {
        id: toastId,
      });
      setProcessing(false);
      return;
    }

    toast.success("Password reset successfully", {
      id: toastId,
    });
    setProcessing(false);
    form.reset();
  }

  return (
    <Form {...form}>
      <form
        onSubmit={form.handleSubmit(onsubmit)}
        className="space-y-4 md:space-y-8"
      >
        <FormField
          control={form.control}
          name="password"
          render={({ field }) => (
            <FormItem>
              <FormLabel>New Password</FormLabel>
              <FormControl>
                <PasswordFormInput processing={processing} field={field} />
              </FormControl>
              <FormMessage>
                {form.formState.errors.password?.message}
              </FormMessage>
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name="confirmPassword"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Confirm New Password</FormLabel>
              <FormControl>
                <PasswordFormInput processing={processing} field={field} />
              </FormControl>
              <FormMessage>
                {form.formState.errors.confirmPassword?.message}
              </FormMessage>
            </FormItem>
          )}
        />
        <Button
          type="submit"
          className="flex w-full justify-center rounded-md px-3 py-1.5 text-sm font-semibold leading-6 "
        >
          Reset Password
        </Button>
      </form>
    </Form>
  );
};

export default ResetPasswordForm;
