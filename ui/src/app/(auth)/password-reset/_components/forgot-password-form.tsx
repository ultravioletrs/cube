"use client";

import { Button } from "@/components/ui/button";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { ResetPasswordRequest } from "@/lib/users";
import { zodResolver } from "@hookform/resolvers/zod";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";
import { z } from "zod";

const formSchema = z.object({
  email: z.string().email({ message: "Invalid email address" }),
});

export const ForgotPasswordForm = () => {
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      email: "",
    },
  });

  const [processing, setProcessing] = useState(false);
  const router = useRouter();

  async function onSubmit(values: z.infer<typeof formSchema>) {
    setProcessing(true);

    const toastId = toast("sonner");
    toast.loading("Sending reset password request...", {
      id: toastId,
    });

    const response = await ResetPasswordRequest(values.email);

    if (response.error) {
      toast.error(response.error, {
        id: toastId,
      });
      setProcessing(false);
      return;
    }

    toast.success(response.data, {
      id: toastId,
    });
    setProcessing(false);
    form.reset();
    router.push("/login");
  }

  return (
    <div className="sm:mx-auto sm:w-full sm:max-w-sm">
      <Form {...form}>
        <form
          onSubmit={form.handleSubmit(onSubmit)}
          className="space-y-4 md:space-y-8"
        >
          <FormField
            control={form.control}
            name="email"
            render={({ field }) => (
              <FormItem>
                <div>
                  Enter your email address and we will send you a link to reset
                  your password.
                </div>
                <FormControl>
                  <Input
                    disabled={processing}
                    placeholder="Enter email"
                    {...field}
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          <Button
            type="submit"
            className="flex w-full justify-center rounded-md bd-indigo-600 px-3 py-1.5 text-sm font-semibold leading-6"
            disabled={processing}
          >
            Send reset link
          </Button>
        </form>
      </Form>
      <div className="my-4 flex items-center">
        <hr className="flex-1 border-gray-300" />
        <span className="mx-4 text-gray-500">OR</span>
        <hr className="flex-1 border-gray-300" />
      </div>
    </div>
  );
};
