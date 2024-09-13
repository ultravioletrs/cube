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
import { Input } from "@/components/ui/input";
import { CreateUser } from "@/lib/users";
import type { User } from "@absmach/magistrala-sdk";
import { zodResolver } from "@hookform/resolvers/zod";
import { signIn } from "next-auth/react";
import { useSearchParams } from "next/navigation";
import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";
import { z } from "zod";

const formSchema = z.object({
  name: z.string().min(1, "Name is requierd"),
  email: z.string().email({ message: "Invalid email address" }),
  password: z.string().min(8, {
    message: "Password must be atleast 8 characters.",
  }),
});

const RegisterForm = () => {
  const [processing, setProcessing] = useState(false);
  const searchParams = useSearchParams();
  const callbackUrl = searchParams.get("callbackUrl");
  const [signInData, setSignInData] = useState<{
    email: string;
    password: string;
  } | null>(null);

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      name: "",
      email: "",
      password: "",
    },
  });

  useEffect(() => {
    if (signInData) {
      toast.promise(
        signIn("credentials", {
          redirect: true,
          callbackUrl: callbackUrl ?? "/",
          email: signInData.email,
          password: signInData.password,
        }),
        {
          loading: "Signing in...",
          duration: 3000,
          success: () => {
            return "Signed in successfully";
          },
          error: () => {
            return "Failed to sign in";
          },
        },
      );
    }
  }, [callbackUrl, signInData]);

  async function onsubmit(values: z.infer<typeof formSchema>) {
    setProcessing(true);

    const user: User = {
      name: values.name,
      credentials: {
        identity: values.email,
        secret: values.password,
      },
      metadata: {
        admin: {},
        ui: {},
      },
    };

    const toastId = toast("Sonner");
    toast.loading("Creating user...", {
      id: toastId,
    });

    const result = await CreateUser(user);

    setProcessing(false);

    if (result.error === null) {
      form.reset();
      setSignInData({ email: values.email, password: values.password });
      toast.success(`User ${result.data} created successfully`, {
        id: toastId,
      });
    } else {
      toast.error(`Failed to create user: ${result.error}`, {
        id: toastId,
      });
    }
  }

  return (
    <Form {...form}>
      <form
        onSubmit={form.handleSubmit(onsubmit)}
        className="space-y-4 md:space-y-8"
      >
        <FormField
          control={form.control}
          name="name"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Name</FormLabel>
              <FormControl>
                <Input
                  placeholder="Enter name "
                  {...field}
                  className="text-popover-foreground"
                />
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
              <FormLabel>Email address</FormLabel>
              <FormControl>
                <Input
                  placeholder="Enter email "
                  disabled={processing}
                  {...field}
                  className="text-popover-foreground"
                />
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
              <FormLabel>Password</FormLabel>
              <FormControl>
                <PasswordFormInput processing={processing} field={field} />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <Button
          type="submit"
          className="flex w-full justify-center rounded-md text-sm  bg-accent text-card-foreground hover:bg-white/85 dark:bg-accent dark:hover:bg-accent/50 dark:hover:text-white hover:text-primaryfont-semibold leading-6 "
          disabled={processing}
        >
          Sign up
        </Button>
      </form>
    </Form>
  );
};

export default RegisterForm;
