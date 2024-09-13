import Breadcrumbs from "@/components/breadcrumbs";
import ErrorComponent from "@/components/error-component";
import { metadataVariables } from "@/constants/metadata-variables";
import { GetDomainPermissions, GetDomainUsers } from "@/lib/domains";
import { getServerSession } from "@/lib/nextauth";
import type { Status, UsersPage } from "@absmach/magistrala-sdk";
import type { Metadata } from "next";
import { AssignMember } from "./_components/assign-member";
import { MembersTable } from "./_components/members-table";

type ParamsProps = {
  params: { slug: string };
  searchParams?: {
    name?: string;
    page: number;
    limit: number;
    status: Status;
  };
};

const baseUrl = `${metadataVariables.baseUrl}/domain/users`;
const title = "Members";
const description =
  "This page allows the user to manage all the members in a domain.";
export const metadata: Metadata = {
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

export default async function Users({ searchParams }: ParamsProps) {
  const page = Number(searchParams?.page) || 1;
  const limit = Number(searchParams?.limit) || 10;
  const name = searchParams?.name || "";
  const status = searchParams?.status || "enabled";
  const response = await GetDomainUsers({
    queryParams: {
      offset: (page > 0 ? page - 1 : 0) * limit,
      limit,
      name: name.trim(),
      status: status,
      // biome-ignore lint/style/useNamingConvention: This is from an external library
      list_perms: true,
    },
  });
  const breadcrumb = [
    { label: "Home", href: "/domain/info" },
    {
      label: "Members",
      href: "/domain/users",
      active: true,
    },
  ];
  const session = await getServerSession();
  const domainPermissions = await GetDomainPermissions(
    session.domain?.id as string,
  );
  let isDomainAdmin = false;
  if (domainPermissions.data) {
    isDomainAdmin = domainPermissions.data.permissions.includes("admin");
  }

  return (
    <div className="container mx-auto mt-4 pb-4 md:pb-8">
      {response.error !== null ? (
        <div className="mt-40">
          <ErrorComponent link="/domain" linkText="Go Back to Homepage" />
        </div>
      ) : (
        <>
          <Breadcrumbs breadcrumbs={breadcrumb} />
          {isDomainAdmin && (
            <div className="flex flex-row justify-end">
              <AssignMember domainId={session.domain?.id as string} />
            </div>
          )}
          <MembersTable
            membersPage={response.data as UsersPage}
            userId={session?.user?.id as string}
            page={page}
            limit={limit}
          />
        </>
      )}
    </div>
  );
}
