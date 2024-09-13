import UpdateMetadataDialog, {
  UpdateTagsDialog,
} from "@/components/entities/update";
import { ViewMetadataDialog } from "@/components/entities/view";
import { Tags } from "@/components/entities/view";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { EntityType, type Metadata } from "@/types/entities";
import type { User } from "@absmach/magistrala-sdk";

export default function ProfileSettings({ user }: { user: User }) {
  return (
    <div className="container mx-auto py-4 md:py-10">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead colSpan={3}>
              Update Your Metadata and Tags here
            </TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          <TableRow>
            <TableHead>Tags</TableHead>
            <TableCell>
              <Tags tags={user.tags} />
            </TableCell>
            <TableCell>
              <UpdateTagsDialog
                id={user.id as string}
                tags={user.tags ? (user.tags as string[]) : []}
                entity={EntityType.User}
              />
            </TableCell>
          </TableRow>
          <TableRow>
            <TableHead>Metadata</TableHead>
            <TableCell>
              <ViewMetadataDialog
                metadata={(user.metadata as Metadata)?.admin}
              />
            </TableCell>
            <TableCell>
              <UpdateMetadataDialog
                id={user.id as string}
                metadata={JSON.stringify(user.metadata) as string}
                entity={EntityType.User}
              />
            </TableCell>
          </TableRow>
        </TableBody>
      </Table>
    </div>
  );
}
