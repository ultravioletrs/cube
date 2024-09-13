import ErrorComponent from "@/components/error-component";
import { Icons } from "@/components/icons";
// Reason for separate component: To make the GetDomains request stream able.
import { GetDomains } from "@/lib/domains";
import type { Domain, Status } from "@absmach/magistrala-sdk";
import HomePageInfinite from "./home-page-infinite";

export default async function HomePageCards({
  params,
}: {
  params: { name: string; status: Status };
}) {
  const response = await GetDomains({
    queryParams: {
      limit: 20,
      name: params.name.trim(),
      status: params.status || "all",
    },
  });

  if (response.error !== null) {
    return <ErrorComponent showLinkButton={false} />;
  }

  const filteredDomains: Domain[] = [];
  if (response?.data) {
    response.data.domains?.map((domain) => {
      if (
        (domain.permission !== "administrator" &&
          domain.status === "enabled") ||
        domain.permission === "administrator"
      ) {
        filteredDomains.push(domain);
      }
    });
  }

  if (filteredDomains && filteredDomains.length > 0) {
    return (
      <HomePageInfinite
        key={Math.random()}
        initDomains={filteredDomains}
        initTotalDomains={response.data.total}
        name={params.name}
      />
    );
  }
  return (
    <div className="flex flex-col item-center justify-center w-full content-center justify-items-center p-4 rounded-2xl bg-accent">
      <Icons.clipboardPen className="h-20 w-20 rounded-full mx-auto pb-2 p-4 border-2 text-muted-foreground" />
      <div className="text24xl text-muted-foreground font-semibold mt-2 text-center">
        No domains found. Get started by creating a new one.
      </div>
    </div>
  );
}
