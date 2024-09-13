import { Button } from "@/components/ui/button";
import {
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { toSentenseCase } from "@/lib/utils";
import { EntityType } from "@/types/entities";
import Link from "next/link";
import { useForm } from "react-hook-form";
interface BulkCreateFormProps {
  entity: EntityType;
  link: string;
  // biome-ignore lint: data is of type any
  onSubmit: (data: any) => Promise<void>;
}

export const BulkCreateForm = ({
  entity,
  link,
  onSubmit,
}: BulkCreateFormProps) => {
  const form = useForm();
  const fileRef = form.register("file");

  return (
    <DialogContent className="rounded max-w-[400px] sm:max-w-[425px]">
      <DialogHeader>
        <DialogTitle>Create {toSentenseCase(entity)}s</DialogTitle>
      </DialogHeader>
      <DialogDescription>
        Add .csv file containing {entity} details. Make sure the following field
        {entity === EntityType.User ? "s are" : " is"} filled in: Name{" "}
        {entity === EntityType.User ? ", Identity and Secret." : "."} The other
        fields can be empty. Find a sample csv file{" "}
        <Link href={link} className="text-blue-500 underline" target="_blank">
          here
        </Link>
        <Form {...form}>
          <form
            onSubmit={form.handleSubmit(onSubmit)}
            className="grid w-full max-w-sm items-center gap-1.5 mt-4"
          >
            <FormField
              control={form.control}
              name="file"
              render={() => {
                return (
                  <FormItem>
                    <FormLabel>.csv file</FormLabel>
                    <FormControl>
                      <Input
                        type="file"
                        accept=".csv"
                        placeholder="Please add a csv file here"
                        {...fileRef}
                      />
                    </FormControl>
                  </FormItem>
                );
              }}
            />
            <Button type="submit">Submit</Button>
          </form>
        </Form>
      </DialogDescription>
    </DialogContent>
  );
};
