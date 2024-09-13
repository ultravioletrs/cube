"use server";
import { GetDomains } from "@/lib/domains";

export async function getDomains({
  token = "",
  page = 1,
  query = "",
}: {
  token: string;
  page: number;
  query: string;
}) {
  const domainPage = await GetDomains({
    token,
    queryParams: {
      name: query.trim(),
      limit: 20,
      offset: (page > 0 ? page - 1 : 0) * 20,
    },
  });
  return domainPage;
}
