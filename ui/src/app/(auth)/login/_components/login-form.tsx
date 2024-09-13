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
import { zodResolver } from "@hookform/resolvers/zod";
import { signIn } from "next-auth/react";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";

const formSchema = z.object({
  email: z.string().email({ message: "Invalid email address" }),
  password: z.string().min(8, {
    message: "Password must be atleast 8 characters.",
  }),
});

const Loginform = () => {
  const [processing, setProcessing] = useState(false);
  const searchParams = useSearchParams();
  const callbackUrl = searchParams.get("callbackUrl");
  const error = searchParams.get("error");
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      email: "",
      password: "",
    },
  });

  async function onsubmit(values: z.infer<typeof formSchema>) {
    setProcessing(true);
    await signIn("credentials", {
      redirect: true,
      callbackUrl: callbackUrl ?? "/",
      email: values.email,
      password: values.password,
    });
    setProcessing(false);
  }
  return (
    <>
      <Form {...form}>
        <form
          onSubmit={form.handleSubmit(onsubmit)}
          className="space-y-4 md:space-y-8"
        >
          <FormField
            control={form.control}
            name="email"
            render={({ field }) => (
              <FormItem>
                <FormLabel>Email address</FormLabel>
                <FormControl>
                  <Input
                    disabled={processing}
                    placeholder="Enter email "
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
                <div className="flex items-center justify-between">
                  <FormLabel>Password</FormLabel>
                  <div className="text-sm">
                    <Link href="/password-reset" className="hover:underline">
                      Forgot password?
                    </Link>
                  </div>
                </div>
                <FormControl>
                  <PasswordFormInput processing={processing} field={field} />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          <Button
            type="submit"
            className="flex w-full justify-center bg-white text-card-foreground hover:bg-white/85 dark:bg-accent dark:hover:bg-accent/50 dark:hover:text-white hover:text-primary rounded-md text-sm font-semibold leading-6"
            disabled={processing}
          >
            Sign In
          </Button>
        </form>
      </Form>
      {error ? (
        <div className="text-md place-content-center mt-2 font-bold text-red-600  ">
          {error}
        </div>
      ) : null}
    </>
  );
};

export default Loginform;
