import ErrorComponent from "@/components/error-component";
import { metadataVariables } from "@/constants/metadata-variables";
import { GetDomains } from "@/lib/domains";
import type { DomainsPage } from "@absmach/magistrala-sdk";
import type { Status } from "@absmach/magistrala-sdk";
import type { Metadata } from "next";
import { CreateDomainForm } from "../../(home)/_components/create-domain";
import { DomainsTable } from "./_components/domains-table";

const baseUrl = `${metadataVariables.baseUrl}/domains`;
const title = "Domains";
const description =
  "This is a list of domains that a user has created or has access to.";

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

const Domains = async ({
  searchParams,
}: {
  searchParams?: {
    name?: string;
    page?: string;
    limit?: string;
    status?: Status;
  };
}) => {
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
    <main className="w-full sm:container pt-20 p-8 md:p-20">
      {response.error !== null ? (
        <div className="mt-40">
          <ErrorComponent link="/" linkText="Go Back to Domain Login" />
        </div>
      ) : (
        <>
          <div className="flex item-center justify-end gap-2">
            <CreateDomainForm />
          </div>
          <DomainsTable
            domainsPage={response.data as DomainsPage}
            limit={limit}
          />
        </>
      )}
    </main>
  );
};

export default Domains;
