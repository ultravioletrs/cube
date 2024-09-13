"use client";

import ErrorComponent from "@/components/error-component";
import { Spinner } from "@/components/ui/spinner";
import { cn } from "@/lib/utils";
import type { Domain } from "@absmach/magistrala-sdk";
import { useSession } from "next-auth/react";
import { useCallback, useEffect, useState } from "react";
import { useInView } from "react-intersection-observer";
import { getDomains } from "../_lib/actions";
import { DomainCard } from "./cards";

export default function HomePageInfinite({
  initDomains,
  initTotalDomains,
  name,
}: {
  initDomains: Domain[];
  initTotalDomains: number;
  name: string | undefined;
}) {
  const [domains, setDomains] = useState(initDomains);
  const [totalDomains, setTotalDomains] = useState(initTotalDomains);
  const [totalLoadedDomains, setTotalLoadedDomains] = useState(
    initDomains.length,
  );
  const [page, setPage] = useState(1);
  const [ref, inView] = useInView();
  const session = useSession();
  const [error, setError] = useState<string | null>(null);

  // biome-ignore lint/correctness/useExhaustiveDependencies: This is a false positive
  const loadMoreDomains = useCallback(async () => {
    const next = page + 1;
    const response = await getDomains({
      token: session.data?.accessToken || "",
      page: next,
      query: name || "",
    });

    if (response.error !== null) {
      setError(response.error);
      return;
    }

    if (response?.data?.domains?.length > 0) {
      setPage(next);
      const filteredDomains: Domain[] = [];
      response.data.domains?.map((domain) => {
        if (
          (domain.permission !== "administrator" &&
            domain.status === "enabled") ||
          domain.permission === "administrator"
        ) {
          filteredDomains.push(domain);
        }
      });
      const newDomains = [...domains, ...filteredDomains];
      setDomains(newDomains);
      setTotalDomains(response.data.total);
      setTotalLoadedDomains(newDomains.length);
    }
  }, [
    setDomains,
    setPage,
    page,
    name,
    session,
    domains,
    setTotalLoadedDomains,
    setTotalDomains,
  ]);

  useEffect(() => {
    if (inView) {
      (async () => {
        await new Promise((resolve) => setTimeout(resolve, 100));
        await loadMoreDomains();
      })();
    }
  }, [inView, loadMoreDomains]);

  return (
    <div className="flex flex-col w-full items-center justify-center">
      {error ? (
        <ErrorComponent showLinkButton={false} />
      ) : (
        <>
          <div
            className={cn(
              "grid grid-flow-row grid-cols-1",
              domains?.length && domains?.length > 1
                ? "sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5"
                : "",
              "gap-4 w-full content-center justify-items-center p-4 rounded-2xl bg-accent",
            )}
          >
            {domains?.length && domains?.length > 0
              ? domains.map((domain) => (
                  <DomainCard key={domain.id} domain={domain} />
                ))
              : null}
          </div>
          {totalLoadedDomains < totalDomains ? (
            <div ref={ref}>
              <Spinner />
            </div>
          ) : null}
        </>
      )}
    </div>
  );
}
