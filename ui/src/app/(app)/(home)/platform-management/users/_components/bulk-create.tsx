"use client";
import { BulkCreateForm } from "@/components/entities/bulk-createform";
import { Button } from "@/components/ui/button";
import { Dialog, DialogTrigger } from "@/components/ui/dialog";
import { CreateUser } from "@/lib/users";
import { EntityType } from "@/types/entities";
import type { User } from "@absmach/magistrala-sdk";
import { Plus } from "lucide-react";
import Papa from "papaparse";
import { useState } from "react";
import { toast } from "sonner";

export function BulkCreate() {
  const [open, setOpen] = useState(false);
  const requiredHeaders = ["Name", "Identity", "Secret", "Tags", "Metadata"];
  // biome-ignore lint: data is of type any
  const onSubmit = async (data: any) => {
    if (data.file && data.file.length > 0) {
      const file = data.file[0];
      Papa.parse(file, {
        header: true,
        skipEmptyLines: true,
        complete: async (results) => {
          const headers = results.meta.fields || [];
          const missingHeaders = requiredHeaders.filter(
            (header) => !headers.includes(header),
          );

          if (missingHeaders.length > 0) {
            toast.error(
              `Invalid file format: Missing required headers: ${missingHeaders.join(
                ", ",
              )}`,
            );
            return;
          }
          // biome-ignore lint: row is of type any
          const parsedResults = results.data.map((row: any, index) => {
            let tags = [];
            let metadata = {};

            if (row.Tags) {
              tags = JSON.parse(row.Tags);
            }

            if (row.Metadata) {
              metadata = JSON.parse(row.Metadata);
            }

            if (!row.Identity || !row.Secret) {
              return {
                faultyRow: index + 1,
                data: null,
              };
            }
            return {
              faultyRow: null,
              data: {
                name: row.Name,
                credentials: {
                  identity: row.Identity,
                  secret: row.Secret,
                },
                metadata: metadata,
                tags: tags,
              } as User,
            };
          });

          const faultyRows = parsedResults.filter(
            (result) => result.faultyRow !== null,
          );

          if (faultyRows.length > 0) {
            const faultyRowNumbers = faultyRows
              .map((result) => result.faultyRow)
              .join(", ");
            toast.error(
              `Invalid file format: Missing Identity or Secret in rows: ${faultyRowNumbers}`,
            );
            return;
          }

          const toastId = toast("Sonner");
          toast.loading("Creating users...", {
            id: toastId,
          });

          for (const result of parsedResults) {
            if (result.data) {
              const user = result.data;
              const createUserResult = await CreateUser(user);
              if (createUserResult.error === null) {
                toast.success("Users created successfully", {
                  id: toastId,
                });
                setOpen(false);
              } else {
                toast.error("Failed to create users", {
                  id: toastId,
                });
              }
            }
          }
        },
      });
    }
  };

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild={true}>
        <Button>
          <Plus className="h-5 mr-2" />
          <span>Create Users</span>
        </Button>
      </DialogTrigger>
      <BulkCreateForm
        entity={EntityType.User}
        link="https://github.com/absmach/magistrala-ui-new/blob/main/samples/users.csv"
        onSubmit={onSubmit}
      />
    </Dialog>
  );
}
