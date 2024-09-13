import Breadcrumbs from "@/components/breadcrumbs";
import { CopyButton } from "@/components/copy";
import { DisplayStatusWithIcon } from "@/components/entities/status-display-with-icon";
import UpdateMetadataDialog, {
  UpdateNameDialog,
  UpdateTagsDialog,
} from "@/components/entities/update";
import { ViewMetadataDialog } from "@/components/entities/view";
import { Tags } from "@/components/entities/view";
import ErrorComponent from "@/components/error-component";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { metadataVariables } from "@/constants/metadata-variables";
import { GetDomainInfo, GetDomainPermissions } from "@/lib/domains";
import { cn } from "@/lib/utils";
import { EntityType, type Metadata } from "@/types/entities";
import type { Status } from "@absmach/magistrala-sdk";
import type { Metadata as NextMetadata } from "next";
import { UpdateAlias, UpdateStatusDialog } from "./_components/update-domain";

const baseUrl = `${metadataVariables.baseUrl}/domain/info`;
const title = "Domain";
const description =
  "This page displays the Domain Information of the current Domain and allows the user to edit the Domain details.";

export const metadata: NextMetadata = {
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
};

export default async function DomainInfo() {
  const response = await GetDomainInfo();

  const breadcrumb = [
    { label: "Home", href: "/domain/info" },
    {
      label: "Domain Info",
      href: "/domain/info",
      active: true,
    },
  ];

  if (!response || response.error || !response.data) {
    return (
      <div className="container mx-auto mt-4 pb-4 md:pb-8">
        <div className="mt-40">
          <ErrorComponent link="/domain" linkText="Go Back to Homepage" />
        </div>
      </div>
    );
  }

  const domain = response.data;
  let canEdit = false;
  const permissions = await GetDomainPermissions(domain.id as string);
  if (permissions.data) {
    canEdit = permissions.data.permissions.includes("edit");
  }

  return (
    <div className="container mx-auto mt-4 pb-4 md:pb-8">
      <Breadcrumbs breadcrumbs={breadcrumb} />
      <Table className="mt-5 bg-white dark:bg-card">
        <TableHeader>
          <TableRow className="hover:bg-transparent">
            <TableHead colSpan={3}>{domain?.name}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          <UpdateNameDialog
            id={domain?.id as string}
            name={domain?.name as string}
            entity={EntityType.Domain}
            canEdit={canEdit}
          />

          <TableRow>
            <TableHead>ID</TableHead>
            <TableCell>
              <span className="me-1">{domain?.id}</span>
              <CopyButton data={domain?.id as string} />
            </TableCell>
          </TableRow>
          {canEdit && (
            <UpdateAlias
              id={domain?.id as string}
              alias={domain?.alias as string}
            />
          )}
          <TableRow>
            <TableHead>Tags</TableHead>
            <TableCell>
              <div
                className={cn("flex flex-row", {
                  "justify-between": domain?.tags && domain?.tags.length > 0,
                  "justify-end": !domain?.tags || domain?.tags.length === 0,
                })}
              >
                <Tags tags={domain?.tags} />
                <div className=" flex flex-row gap-4">
                  {canEdit && (
                    <UpdateTagsDialog
                      id={domain?.id as string}
                      tags={domain?.tags ? (domain?.tags as string[]) : []}
                      entity={EntityType.Domain}
                    />
                  )}
                </div>
              </div>
            </TableCell>
          </TableRow>
          <TableRow>
            <TableHead>Metadata</TableHead>
            <TableCell>
              <div className=" flex flex-row justify-between">
                <ViewMetadataDialog metadata={domain?.metadata as Metadata} />
                <div className="flex flex-row gap-4">
                  {canEdit && (
                    <UpdateMetadataDialog
                      id={domain?.id as string}
                      metadata={JSON.stringify(domain?.metadata) as string}
                      entity={EntityType.Domain}
                    />
                  )}
                </div>
              </div>
            </TableCell>
          </TableRow>
          <TableRow>
            <TableHead>Status</TableHead>
            <TableCell>
              <div className="flex flex-row justify-between">
                <DisplayStatusWithIcon status={domain?.status as string} />
                <div className="flex">
                  {permissions.data?.permissions.includes("admin") && (
                    <UpdateStatusDialog
                      status={domain?.status as Status}
                      domain={domain}
                    />
                  )}
                </div>
              </div>
            </TableCell>
          </TableRow>
        </TableBody>
      </Table>
    </div>
  );
}
