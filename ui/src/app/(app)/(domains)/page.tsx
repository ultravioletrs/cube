import SearchInput from "@/components/tables/search";
import Statusbutton from "@/components/tables/status-button";
import { metadataVariables } from "@/constants/metadata-variables";
import type { Status } from "@absmach/magistrala-sdk";
import type { Metadata } from "next";
import { Suspense } from "react";
import CardsSkeleton from "../(home)/_components/cards-skeleton";
import { CreateDomainForm } from "../(home)/_components/create-domain";
import HomePageCards from "../(home)/_components/home-page-cards";

const baseUrl = `${metadataVariables.baseUrl}`;
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

type ParamsProps = {
  searchParams?: {
    name?: string;
    status: Status;
  };
};

export default function Home({ searchParams }: ParamsProps) {
  const name = searchParams?.name || "";
  const status = searchParams?.status || "enabled";
  const suspenseKey = `search=${name}+${Math.random()}`;
  return (
    <>
      <div className="flex item-center justify-center">
        <main className="container py-24 flex flex-col min-h-screen  gap-4">
          <div className="flex flex-col md:flex-row items-center gap-8">
            <div className="w-full">
              <SearchInput placeholder="Search Domain" canFilter={false} />
            </div>
            <div className="flex gap-2 justify-end">
              <Statusbutton />
              <CreateDomainForm />
            </div>
          </div>
          <Suspense key={suspenseKey} fallback={<CardsSkeleton />}>
            <HomePageCards
              params={{
                name,
                status,
              }}
            />
          </Suspense>
        </main>
      </div>
    </>
  );
}
