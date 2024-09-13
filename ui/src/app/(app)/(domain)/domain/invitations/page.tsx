import Breadcrumbs from "@/components/breadcrumbs";
import ErrorComponent from "@/components/error-component";
import { metadataVariables } from "@/constants/metadata-variables";
import { GetInvitations } from "@/lib/invitations";
import { getServerSession } from "@/lib/nextauth";
import type { InvitationsPage } from "@absmach/magistrala-sdk";
import type { Metadata } from "next";
import { InvitationsTable } from "./_components/invitations-table";
import { SendInvitationForm } from "./_components/sendinvitation-form";

type ParamsProps = {
  params: { slug: string };
  searchParams?: { name?: string; page: number; limit: number };
};

const baseUrl = `${metadataVariables.baseUrl}/domain/invitations`;
const title = "Invitations";
const description =
  "This page contains invitations sent to users and allows domain administrators to send invitations to users to join the domain.";

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

export default async function Invitations({ searchParams }: ParamsProps) {
  const page = Number(searchParams?.page) || 1;
  const limit = Number(searchParams?.limit) || 10;
  const session = await getServerSession();
  const response = await GetInvitations({
    queryParams: {
      offset: (page > 0 ? page - 1 : 0) * limit,
      limit,
      // biome-ignore lint/style/useNamingConvention: This is from an external library
      domain_id: session?.domain?.id,
      state: "pending",
    },
  });
  const breadcrumb = [
    { label: "Home", href: "/domain/info" },
    {
      label: "Invitations",
      href: "/domain/invitations",
      active: true,
    },
  ];
  return (
    <div className="container mx-auto mt-4 pb-4 md:pb-8">
      {response.error !== null ? (
        <div className="mt-40">
          <ErrorComponent link="/domain" linkText="Go Back to Homepage" />
        </div>
      ) : (
        <>
          <Breadcrumbs breadcrumbs={breadcrumb} />
          <div className="flex item-center justify-end gap-2">
            <SendInvitationForm id={session.domain?.id as string} />
          </div>
          <InvitationsTable
            invitationsPage={response.data as InvitationsPage}
            page={page}
            limit={limit}
          />
        </>
      )}
    </div>
  );
}
