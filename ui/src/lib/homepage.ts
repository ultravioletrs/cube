import type { Icons } from "@/components/icons";
import { getServerSession } from "@/lib/nextauth";
import type { HttpError } from "@/types/errors";
import { mgSdk, validateOrGetToken } from "./magistrala";

export interface EntitiesData {
  total: number;
  name: string;
  enabled: number;
  disabled: number;
  icon: keyof typeof Icons;
  error?: string;
}

export interface HomePageData {
  entitiesData: EntitiesData[];
  error: string | null;
}

export const GetHomePageData = async (): Promise<HomePageData> => {
  try {
    const accessToken = await validateOrGetToken("");
    const session = await getServerSession();
    const domainId = session.domain?.id as string;
    const domainMembers: EntitiesData = {
      total: 0,
      name: "Domain Members",
      enabled: 0,
      disabled: 0,
      icon: "users",
    };

    try {
      const allUsers = await mgSdk.domains.ListDomainUsers(
        domainId,
        {
          offset: 0,
          limit: 10,
          status: "all",
        },
        accessToken,
      );

      const enabledUsers = await mgSdk.domains.ListDomainUsers(
        domainId,
        {
          offset: 0,
          limit: 10,
          status: "enabled",
        },
        accessToken,
      );

      const disabledUsers = await mgSdk.domains.ListDomainUsers(
        domainId,
        {
          offset: 0,
          limit: 10,
          status: "disabled",
        },
        accessToken,
      );

      domainMembers.total = allUsers.total;
      domainMembers.enabled = enabledUsers.total;
      domainMembers.disabled = disabledUsers.total;
    } catch (err: unknown) {
      const knownError = err as HttpError;
      domainMembers.error =
        knownError.error || knownError.message || knownError.toString();
    }

    const homePageData: HomePageData = {
      entitiesData: [domainMembers],
      error: null,
    };

    return homePageData;
  } catch (err: unknown) {
    const knownError = err as HttpError;
    return {
      entitiesData: [],
      error: knownError.error || knownError.message || knownError.toString(),
    };
  }
};
