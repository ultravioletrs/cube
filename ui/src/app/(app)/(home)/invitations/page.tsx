import ErrorComponent from "@/components/error-component";
import { metadataVariables } from "@/constants/metadata-variables";
import { GetInvitations } from "@/lib/invitations";
import { getServerSession } from "@/lib/nextauth";
import type { InvitationsPage } from "@absmach/magistrala-sdk";
import type { Metadata } from "next";
import { InvitationsTable } from "./_components/invitations-page";

type ParamsProps = {
  params: { slug: string };
  searchParams?: { name?: string; page: number; limit: number };
};

const baseUrl = `${metadataVariables.baseUrl}/invitations}`;
const title = "Invitations";
const description =
  "This page allows the user to manage their pending invitations.";
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
      user_id: session?.user?.id,
      state: "pending",
    },
  });
  return (
    <div className="container mx-auto py-10 md:py-10 pb-10">
      {response.error !== null ? (
        <div className="mt-40">
          <ErrorComponent link="/" linkText="Go Back to Domain Login" />
        </div>
      ) : (
        <>
          <div className="mt-4 flex item-center justify-end gap-2 md:mt-8" />
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
