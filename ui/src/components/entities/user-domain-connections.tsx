import { RequiredAsterisk } from "@/components/required";
import { Button } from "@/components/ui/button";
import {
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import type { EntityFetchData } from "@/lib/actions";
import {
  AssignMultipleUsersToDomain,
  AssignUserToMultipleDomains,
  UnassignUserFromDomain,
} from "@/lib/domains";
import {
  InviteMultipleUsersToDomain,
  InviteUserToMultipleDomains,
} from "@/lib/invitations";
import { DomainRelations, EntityType } from "@/types/entities";
import type { Relation } from "@absmach/magistrala-sdk";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { toast } from "sonner";
import { z } from "zod";
import { MultipleSelect } from "./multiple-select";
import UserSearchInput from "./user-search-input";

const inviteMultipleUsersToDomainFormSchema = z.object({
  userIds: z
    .string()
    .array()
    .nonempty({ message: "Please select at least one user" }),
  // biome-ignore lint/style/useNamingConvention: This is from an external library
  relation: z.string({ required_error: "Please select a relation" }),
  domainId: z.string(),
});

export const InviteMultipleUsersDialog = ({
  id,
  setOpen,
}: {
  setOpen: (show: boolean) => void;
  id: string;
}) => {
  const form = useForm<z.infer<typeof inviteMultipleUsersToDomainFormSchema>>({
    resolver: zodResolver(inviteMultipleUsersToDomainFormSchema),
    defaultValues: {
      domainId: id,
    },
  });

  async function onSubmit(
    values: z.infer<typeof inviteMultipleUsersToDomainFormSchema>,
  ) {
    const toastId = toast("Sonner");

    toast.loading("Sending Invitation(s)...", {
      id: toastId,
    });

    const result = await InviteMultipleUsersToDomain(
      values.userIds,
      values.domainId,
      values.relation as Relation,
    );
    if (result.errors.length === 0) {
      toast.success(
        `User(s) ${values.userIds} invited to domain ${id} with relation ${values.relation} successfully.`,
        {
          id: toastId,
        },
      );
      form.reset();
      setOpen(false);
    } else {
      const errorMessages = result.errors
        .map((errorObj) => errorObj.error)
        .join(", ");
      toast.error(`Failed to send invitation to user(s): ${errorMessages}`, {
        id: toastId,
      });
    }
  }
  return (
    <DialogContent className="rounded max-w-[400px] sm:max-w-[425px]">
      <DialogHeader>
        <DialogTitle>Send Invitation(s)</DialogTitle>
      </DialogHeader>
      <Form {...form}>
        <form
          onSubmit={form.handleSubmit(onSubmit)}
          className="space-y-4 md:space-y-8"
        >
          <FormField
            control={form.control}
            name="userIds"
            render={({ field }) => (
              <FormItem>
                <FormLabel>
                  User(s) <RequiredAsterisk />
                </FormLabel>
                <FormControl>
                  <UserSearchInput field={field} />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          <Input type="hidden" name="domainId" defaultValue={id} />
          <FormField
            control={form.control}
            name="relation"
            render={({ field }) => (
              <FormItem>
                <FormLabel>
                  Relation <RequiredAsterisk />
                </FormLabel>
                <Select
                  onValueChange={field.onChange}
                  defaultValue={field.value}
                >
                  <FormControl>
                    <SelectTrigger>
                      <SelectValue placeholder="Select relation" />
                    </SelectTrigger>
                  </FormControl>
                  <SelectContent>
                    {DomainRelations.map((relation) => (
                      <SelectItem key={relation.toString()} value={relation}>
                        {relation}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
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
                }}
              >
                Close
              </Button>
            </DialogClose>
            <Button type="submit">Send</Button>
          </DialogFooter>
        </form>
      </Form>
    </DialogContent>
  );
};

const inviteUserToMultipleDomainsFormSchema = z.object({
  // biome-ignore lint/style/useNamingConvention: This is from an external library
  userId: z.string({ required_error: "Please select a member to add" }),
  // biome-ignore lint/style/useNamingConvention: This is from an external library
  relation: z.string({ required_error: "Please select a relation" }),
  domainIds: z
    .string()
    .array()
    .nonempty({ message: "Please select at least one domain" }),
});

export const InviteUserToMultipleDomainsDialog = ({
  initData,
  id,
  setOpen,
}: {
  initData: EntityFetchData;
  setOpen: (show: boolean) => void;
  id: string;
}) => {
  const form = useForm<z.infer<typeof inviteUserToMultipleDomainsFormSchema>>({
    resolver: zodResolver(inviteUserToMultipleDomainsFormSchema),
    defaultValues: {
      domainIds: [],
      userId: id,
    },
  });

  async function onSubmit(
    values: z.infer<typeof inviteUserToMultipleDomainsFormSchema>,
  ) {
    const toastId = toast("Sonner");

    toast.loading("Sending Invitation(s)...", {
      id: toastId,
    });

    const result = await InviteUserToMultipleDomains(
      values.domainIds,
      values.userId,
      values.relation as Relation,
    );
    if (result.errors.length === 0) {
      toast.success(
        `User ${id} invited to domain(s) ${values.domainIds} with relation ${values.relation} successfully.`,
        {
          id: toastId,
        },
      );
      form.reset();
      setOpen(false);
    } else {
      const errorMessages = result.errors
        .map((errorObj) => errorObj.error)
        .join(", ");
      toast.error(`Failed to send invitation(s) to user: ${errorMessages}`, {
        id: toastId,
      });
    }
  }
  return (
    <DialogContent className="rounded max-w-[400px] sm:max-w-[425px]">
      <DialogHeader>
        <DialogTitle>Send Invitation(s)</DialogTitle>
      </DialogHeader>
      <Form {...form}>
        <form
          onSubmit={form.handleSubmit(onSubmit)}
          className="space-y-4 md:space-y-8"
        >
          <FormField
            control={form.control}
            name="domainIds"
            render={({ field }) => (
              <FormItem>
                <FormLabel>
                  Domains <RequiredAsterisk />
                </FormLabel>
                <FormControl>
                  <MultipleSelect
                    field={field}
                    entityType={EntityType.Domain}
                    initData={initData}
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          <Input type="hidden" name="userId" defaultValue={id} />

          <FormField
            control={form.control}
            name="relation"
            render={({ field }) => (
              <FormItem>
                <FormLabel>
                  Relation <RequiredAsterisk />
                </FormLabel>
                <Select
                  onValueChange={field.onChange}
                  defaultValue={field.value}
                >
                  <FormControl>
                    <SelectTrigger>
                      <SelectValue placeholder="Select relation" />
                    </SelectTrigger>
                  </FormControl>
                  <SelectContent>
                    {DomainRelations.map((relation) => (
                      <SelectItem key={relation.toString()} value={relation}>
                        {relation}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
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
                }}
              >
                Close
              </Button>
            </DialogClose>
            <Button type="submit">Send</Button>
          </DialogFooter>
        </form>
      </Form>
    </DialogContent>
  );
};

const assignMultipleUsersToDomainFormSchema = z.object({
  userIds: z
    .string()
    .array()
    .nonempty({ message: "Please select at least one user" }),
  domainId: z.string(),
  // biome-ignore lint/style/useNamingConvention: This is from an external library
  relation: z.string({ required_error: "Please select a relation" }),
});

export const AssignMultipleUsersToDomainDialog = ({
  setOpen,
  domainId,
}: {
  setOpen: (show: boolean) => void;
  domainId: string;
}) => {
  const form = useForm<z.infer<typeof assignMultipleUsersToDomainFormSchema>>({
    resolver: zodResolver(assignMultipleUsersToDomainFormSchema),
    defaultValues: {
      domainId: domainId,
    },
  });
  async function onSubmit(
    values: z.infer<typeof assignMultipleUsersToDomainFormSchema>,
  ) {
    const toastId = toast("Sonner");
    toast.loading("Assigning user to domain...", {
      id: toastId,
    });
    const result = await AssignMultipleUsersToDomain(
      values.userIds,
      values.domainId,
      values.relation as Relation,
    );
    if (result.error === null) {
      toast.success(
        `User(s) ${values.userIds} successfully assigned to domain ${values.domainId}`,
        {
          id: toastId,
        },
      );
      form.reset();
      setOpen(false);
    } else {
      toast.error(`Failed to assign user to domain: ${result.error}`, {
        id: toastId,
      });
    }
  }
  return (
    <DialogContent className="rounded max-w-[400px] sm:max-w-[425px]">
      <DialogHeader>
        <DialogTitle>Assign User(s)</DialogTitle>
      </DialogHeader>
      <Form {...form}>
        <form
          onSubmit={form.handleSubmit(onSubmit)}
          className="space-y-4 md:space-y-8"
        >
          <FormField
            control={form.control}
            name="userIds"
            render={({ field }) => (
              <FormItem>
                <FormLabel>
                  User(s) <RequiredAsterisk />
                </FormLabel>
                <FormControl>
                  <UserSearchInput field={field} />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          <Input type="hidden" name="domainId" defaultValue={domainId} />
          <FormField
            control={form.control}
            name="relation"
            render={({ field }) => (
              <FormItem>
                <FormLabel>
                  Relation <RequiredAsterisk />
                </FormLabel>
                <Select
                  onValueChange={field.onChange}
                  defaultValue={field.value}
                >
                  <FormControl>
                    <SelectTrigger>
                      <SelectValue placeholder="Select relation" />
                    </SelectTrigger>
                  </FormControl>
                  <SelectContent>
                    {DomainRelations.map((relation) => (
                      <SelectItem key={relation.toString()} value={relation}>
                        {relation}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FormItem>
            )}
          />
          <DialogFooter className="flex flex-row justify-end gap-2">
            <DialogClose asChild={true}>
              <Button type="button" variant="secondary">
                Close
              </Button>
            </DialogClose>
            <Button type="submit">Assign</Button>
          </DialogFooter>
        </form>
      </Form>
    </DialogContent>
  );
};

const AssignUserToMultipleDomainsFormSchema = z.object({
  domainIds: z
    .string()
    .array()
    .nonempty({ message: "Please select at least one domain" }),
  // biome-ignore lint/style/useNamingConvention: This is from an external library
  relation: z.string({ required_error: "Please select a relation" }),
  userId: z.string(),
});

export const AssignUserToMultipleDomainsDialog = ({
  initData,
  setOpen,
  userId,
}: {
  initData: EntityFetchData;
  setOpen: (show: boolean) => void;
  userId: string;
}) => {
  const form = useForm<z.infer<typeof AssignUserToMultipleDomainsFormSchema>>({
    resolver: zodResolver(AssignUserToMultipleDomainsFormSchema),
    defaultValues: {
      userId: userId,
      domainIds: [],
    },
  });
  async function onSubmit(
    values: z.infer<typeof AssignUserToMultipleDomainsFormSchema>,
  ) {
    const toastId = toast("Sonner");
    toast.loading("Assigning user to domain(s)...", {
      id: toastId,
    });
    const result = await AssignUserToMultipleDomains(
      values.domainIds,
      values.userId,
      values.relation as Relation,
    );

    if (result.errors.length === 0) {
      toast.success(
        `User ${userId} successfully assigned to domain(s) ${values.domainIds}`,
        {
          id: toastId,
        },
      );
      form.reset();
      setOpen(false);
    } else {
      const errorMessages = result.errors
        .map((errorObj) => errorObj.error)
        .join(", ");
      toast.error(`Failed to assign user to domain(s): ${errorMessages}`, {
        id: toastId,
      });
    }
  }
  return (
    <DialogContent className="rounded max-w-[400px] sm:max-w-[425px]">
      <DialogHeader>
        <DialogTitle>Assign User</DialogTitle>
      </DialogHeader>
      <Form {...form}>
        <form
          onSubmit={form.handleSubmit(onSubmit)}
          className="space-y-4 md:space-y-8"
        >
          <FormField
            control={form.control}
            name="domainIds"
            render={({ field }) => (
              <FormItem>
                <FormLabel>
                  Domain(s) <RequiredAsterisk />
                </FormLabel>
                <FormControl>
                  <MultipleSelect
                    field={field}
                    entityType={EntityType.Domain}
                    initData={initData}
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          <Input type="hidden" name="userId" defaultValue={userId} />

          <FormField
            control={form.control}
            name="relation"
            render={({ field }) => (
              <FormItem>
                <FormLabel>
                  Relation <RequiredAsterisk />
                </FormLabel>
                <Select
                  onValueChange={field.onChange}
                  defaultValue={field.value}
                >
                  <FormControl>
                    <SelectTrigger>
                      <SelectValue placeholder="Select relation" />
                    </SelectTrigger>
                  </FormControl>
                  <SelectContent>
                    {DomainRelations.map((relation) => (
                      <SelectItem key={relation.toString()} value={relation}>
                        {relation}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FormItem>
            )}
          />
          <DialogFooter className="flex flex-row justify-end gap-2">
            <DialogClose asChild={true}>
              <Button type="button" variant="secondary">
                Close
              </Button>
            </DialogClose>
            <Button type="submit">Assign</Button>
          </DialogFooter>
        </form>
      </Form>
    </DialogContent>
  );
};

export const UnassignUserFromDomainDialog = ({
  setOpen,
  userId,
}: {
  setOpen: (show: boolean) => void;
  userId: string;
}) => {
  async function handleAction() {
    const toastId = toast("Sonner");
    toast.loading("Unassigning user from domain...", {
      id: toastId,
    });
    const response = await UnassignUserFromDomain(userId);

    if (response?.error === null) {
      toast.success(`User ${userId} successfully unassigned user from domain`, {
        id: toastId,
      });
      setOpen(false);
    } else {
      toast.error(`Failed to unassign user from domain: ${response?.error}`, {
        id: toastId,
      });
    }
  }
  return (
    <DialogContent className="rounded max-w-[400px] sm:max-w-[425px]">
      <DialogHeader>
        <DialogTitle>Unassign User</DialogTitle>
      </DialogHeader>
      <DialogDescription>
        Are you sure you want to <span className="text-red-500">unassign</span>{" "}
        user from domain ?
      </DialogDescription>
      <DialogFooter>
        <DialogClose asChild={true}>
          <Button type="button" variant="secondary">
            Close
          </Button>
        </DialogClose>
        <Button type="button" onClick={handleAction} variant="destructive">
          Unassign
        </Button>
      </DialogFooter>
    </DialogContent>
  );
};
