"use client";
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
import { Input } from "@/components/ui/input";
import { UpdateServerSession } from "@/lib/actions";
import { UpdateUser, UpdateUserIdentity } from "@/lib/users";
import type { Session } from "@/types/auth";
import type { User } from "@absmach/magistrala-sdk";
import { zodResolver } from "@hookform/resolvers/zod";
import { signOut, useSession } from "next-auth/react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";
import { z } from "zod";

const formSchema = z.object({
  email: z.string().email({
    message: "Please enter a valid email address",
  }),
  // biome-ignore lint/style/useNamingConvention: This is from an external library
  username: z.string({ required_error: "User name is required" }),
});

export default function ProfileForm({
  serverSession,
}: {
  serverSession: Session;
}) {
  const { data: session, update } = useSession();
  const initialEmail = serverSession?.user?.email;
  const initialUsername = serverSession?.user?.name;
  const id = serverSession?.user?.id;
  const loginUrl = `${process.env.NEXT_PUBLIC_BASE_URL}/login`;

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      email: initialEmail ?? "",
      username: initialUsername ?? "",
    },
  });

  async function onSubmit(values: z.infer<typeof formSchema>) {
    const user: User = {
      id: id,
      credentials: {
        identity: values.email,
      },
      name: values.username,
    };
    const caseType =
      values.email !== initialEmail && values.username !== initialUsername
        ? "both"
        : values.username !== initialUsername
          ? "username"
          : "email";

    switch (caseType) {
      case "both": {
        toast.promise(
          (async () => {
            await UpdateUser(user);
            await UpdateServerSession();
            await UpdateUserIdentity(user);
          })(),
          {
            loading: "Updating Account Details...",
            duration: 7000,
            success: () => {
              signOut({
                callbackUrl: loginUrl,
              });
              return "Account Details updated successfully";
            },
            error: () => {
              return "Failed to update Account Details";
            },
          },
        );
        break;
      }
      case "username": {
        const formData = new FormData();
        formData.append("username", values.username);
        toast.promise(UpdateUser(user), {
          loading: "Updating username...",
          duration: 7000,
          success: () => {
            form.reset();
            form.reset({ username: values.username });
            UpdateServerSession();
            update({ ...session?.user, name: values.username });
            return "Username updated successfully.";
          },
          error: () => {
            return "Failed to update username";
          },
        });
        break;
      }
      case "email": {
        toast.promise(UpdateUserIdentity(user), {
          loading: "Updating email...",
          duration: 7000,
          success: () => {
            signOut({
              callbackUrl: loginUrl,
            });
            return "Email updated successfully";
          },
          error: () => {
            return "Failed to update email";
          },
        });
        break;
      }
    }
  }

  return (
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
              <FormLabel>
                Email <RequiredAsterisk />
              </FormLabel>
              <FormControl>
                <Input {...field} type="email" placeholder="Enter your email" />
              </FormControl>
              <FormMessage {...field} />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name="username"
          render={({ field }) => (
            <FormItem>
              <FormLabel>
                User Name <RequiredAsterisk />
              </FormLabel>
              <FormControl>
                <Input
                  {...field}
                  type="username"
                  placeholder="Enter your username"
                />
              </FormControl>
              <FormMessage {...field} />
            </FormItem>
          )}
        />
        <Button type="submit" disabled={form.formState.isSubmitting}>
          Update
        </Button>
      </form>
    </Form>
  );
}
