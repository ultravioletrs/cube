import ErrorComponent from "@/components/error-component";
import { GetDomains } from "@/lib/domains";
import type { DomainsPage, Status } from "@absmach/magistrala-sdk";
import { DomainsTable } from "./_components/domains-table";

type Props = {
  searchParams?: {
    name?: string;
    page?: string;
    limit?: string;
    status?: Status;
  };
};
export default async function PlatformManagement({ searchParams }: Props) {
  const page = Number(searchParams?.page) || 1;
  const limit = Number(searchParams?.limit) || 10;
  const name = searchParams?.name || "";
  const status = searchParams?.status || "enabled";
  const response = await GetDomains({
    queryParams: {
      offset: (page > 0 ? page - 1 : 0) * limit,
      limit,
      name: name.trim(),
      status: status,
    },
  });

  return (
    <>
      {response.error !== null ? (
        <ErrorComponent
          link="/platform-management"
          linkText="Go Back to Domain Login"
        />
      ) : (
        <DomainsTable
          domainsPage={response.data as DomainsPage}
          limit={limit}
        />
      )}
    </>
  );
}
