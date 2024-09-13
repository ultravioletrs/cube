import { CopyButton } from "@/components/copy";
import { DisplayStatusWithIcon } from "@/components/entities/status-display-with-icon";
import UpdateMetadataDialog, {
  UpdateNameDialog,
  UpdateTagsDialog,
} from "@/components/entities/update";
import { ViewMetadataDialog } from "@/components/entities/view";
import { Tags } from "@/components/entities/view";
import ErrorComponent from "@/components/error-component";
import StatusSwitch from "@/components/tables/status-switch";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
} from "@/components/ui/table";
import { metadataVariables } from "@/constants/metadata-variables";
import { ViewUser } from "@/lib/users";
import { cn } from "@/lib/utils";
import { EntityType, type Metadata } from "@/types/entities";
import type { Status } from "@absmach/magistrala-sdk";
import { User2 } from "lucide-react";
import { UpdateEmail, UpdateRole } from "./_components/update-user";

type ParamsProps = {
  params: { id: string };
};

export const generateMetadata = ({
  params,
}: ParamsProps): Promise<Metadata> => {
  const baseUrl = `${metadataVariables.baseUrl}/users/${params.id}`;
  const title = "User";
  const description =
    "This page allows the platform administrator to manage all user settings.";
  return Promise.resolve({
    metadataBase: new URL(baseUrl),
    title: title,
    description: description,
    openGraph: {
      title: title,
      description: description,
      url: baseUrl,
      type: "website",
      images: metadataVariables.image,
    },
  });
};

export default async function User({ params }: ParamsProps) {
  const response = await ViewUser(params.id);
  const user = response.data;

  return (
    <>
      {response.error !== null ? (
        <div className="mt-40">
          <ErrorComponent link="/users" linkText="Go Back to Users Page" />
        </div>
      ) : (
        <>
          <div className="grid grid-rows-2 gap-4 mt-8">
            <div className="border-b pb-4 sm:border sm:p-4">
              <h2 className="text-2xl font-bold flex flex-row gap-2 items-center">
                <User2 />
                <span>{`${user?.name as string} Details`}</span>
              </h2>
              <div className="border rounded-md mt-8">
                <Table>
                  <TableBody>
                    <UpdateNameDialog
                      id={user?.id as string}
                      name={user?.name as string}
                      entity={EntityType.User}
                    />
                    <TableRow>
                      <TableHead>ID</TableHead>
                      <TableCell>
                        <span className="me-1">{user?.id}</span>
                        <CopyButton data={user?.id as string} />
                      </TableCell>
                    </TableRow>
                    <UpdateEmail
                      id={user?.id as string}
                      email={user?.credentials?.identity as string}
                    />
                    <TableRow>
                      <TableHead>Tags</TableHead>
                      <TableCell>
                        <div
                          className={cn("flex flex-row items-center", {
                            "justify-between":
                              user?.tags && user?.tags.length > 0,
                            "justify-end":
                              !user?.tags || user?.tags.length === 0,
                          })}
                        >
                          <Tags tags={user?.tags} />
                          <div className=" flex flex-row gap-4">
                            <UpdateTagsDialog
                              id={user?.id as string}
                              tags={user?.tags ? (user?.tags as string[]) : []}
                              entity={EntityType.User}
                            />
                          </div>
                        </div>
                      </TableCell>
                    </TableRow>
                    <TableRow>
                      <TableHead>Metadata</TableHead>
                      <TableCell>
                        <div className=" flex flex-row justify-between">
                          <ViewMetadataDialog
                            metadata={user?.metadata as Metadata}
                          />
                          <div className="flex flex-row gap-4">
                            <UpdateMetadataDialog
                              id={user?.id as string}
                              metadata={
                                JSON.stringify(user?.metadata) as string
                              }
                              entity={EntityType.User}
                            />
                          </div>
                        </div>
                      </TableCell>
                    </TableRow>
                    <TableRow>
                      <TableHead>Role</TableHead>
                      <TableCell>
                        <div className="flex flex-row justify-between items-center">
                          <span>{user?.role ? user?.role : "user"}</span>
                          <UpdateRole
                            id={user?.id as string}
                            name={user?.name as string}
                            role={user?.role as string}
                          />
                        </div>
                      </TableCell>
                    </TableRow>
                    <TableRow>
                      <TableHead>Status</TableHead>
                      <TableCell>
                        <div className=" flex flex-row justify-between">
                          <DisplayStatusWithIcon
                            status={user?.status as string}
                          />
                          <div className="flex flex-row gap-4">
                            <StatusSwitch
                              entity={EntityType.User}
                              status={user?.status as Status}
                              id={user?.id as string}
                            />
                          </div>
                        </div>
                      </TableCell>
                    </TableRow>
                  </TableBody>
                </Table>
              </div>
            </div>
          </div>
        </>
      )}
    </>
  );
}
