"use server";
import { getServerSession } from "@/lib/nextauth";
import type { HttpError } from "@/types/errors";
import type {
  Domain,
  DomainBasicInfo,
  DomainsPage,
  Relation,
  UserBasicInfo,
} from "@absmach/magistrala-sdk";
import { revalidatePath } from "next/cache";
import { type RequestOptions, mgSdk, validateOrGetToken } from "./magistrala";

export const CreateDomain = async (domain: Domain) => {
  const session = await getServerSession();
  const accessToken = session?.accessToken;
  try {
    const created = await mgSdk.domains.CreateDomain(domain, accessToken);
    return {
      data: created.name as string,
      error: null,
    };
  } catch (err: unknown) {
    const knownError = err as HttpError;
    return {
      data: null,
      error: knownError.error || knownError.message || knownError.toString(),
    };
  } finally {
    revalidatePath("/domains");
  }
};

export const GetDomains = async ({
  token = "",
  queryParams,
}: RequestOptions) => {
  try {
    const accessToken = await validateOrGetToken(token);
    const domainsPage = await mgSdk.domains.Domains(queryParams, accessToken);
    const domains = await processDomains(domainsPage.domains, accessToken);
    return {
      data: {
        total: domainsPage.total,
        offset: domainsPage.offset,
        limit: domainsPage.limit,
        domains,
      } as DomainsPage,
      error: null,
    };
  } catch (err: unknown) {
    const knownError = err as HttpError;
    return {
      data: null,
      error: knownError.error || knownError.message || knownError.toString(),
    };
  }
};

export const GetDomainUsers = async ({
  token = "",
  queryParams,
}: RequestOptions) => {
  try {
    const accessToken = await validateOrGetToken(token);
    const session = await getServerSession();
    const domainUsers = await mgSdk.domains.ListDomainUsers(
      session.domain?.id as string,
      queryParams,
      accessToken,
    );

    return {
      data: domainUsers,
      error: null,
    };
  } catch (err: unknown) {
    const knownError = err as HttpError;
    return {
      data: null,
      error: knownError.error || knownError.message || knownError.toString(),
    };
  }
};

async function processDomains(
  domains: Domain[],
  token: string,
): Promise<Domain[]> {
  const processedDomains: Domain[] = [];
  if (domains && domains.length > 0) {
    for (const domain of domains) {
      try {
        const createdBy: UserBasicInfo | string =
          typeof domain.created_by === "string"
            ? domain.created_by === ""
              ? domain.created_by
              : await GetUserBasicInfo(domain.created_by, token)
            : (domain.created_by as UserBasicInfo);

        const updatedBy: UserBasicInfo | string =
          typeof domain.updated_by === "string"
            ? domain.updated_by === ""
              ? domain.updated_by
              : await GetUserBasicInfo(domain.updated_by, token)
            : (domain.updated_by as UserBasicInfo);

        const processedDomain: Domain = {
          ...domain,
          // biome-ignore lint/style/useNamingConvention: This is from an external library
          created_by: createdBy,
          // biome-ignore lint/style/useNamingConvention: This is from an external library
          updated_by: updatedBy,
        };
        processedDomains.push(processedDomain);
      } catch {
        processedDomains.push(domain);
      }
    }
    return processedDomains;
  }
  return domains;
}

export async function GetUserBasicInfo(userId: string, token = "") {
  try {
    const accessToken = await validateOrGetToken(token);
    const userInfo = await mgSdk.users.User(userId, accessToken);
    return {
      id: userInfo.id,
      name: userInfo.name,
      status: userInfo.status,
      credentials: userInfo.credentials,
    } as UserBasicInfo;
  } catch (_error) {
    return userId;
  }
}

export const GetDomainInfo = async () => {
  try {
    const session = await getServerSession();
    if (session.domain?.id && session.domain?.id !== "") {
      const domain = await mgSdk.domains.Domain(
        session.domain.id,
        session.accessToken,
      );
      return {
        data: domain,
        error: null,
      };
    }
  } catch (err: unknown) {
    const knownError = err as HttpError;
    return {
      data: null,
      error: knownError.error || knownError.message || knownError.toString(),
    };
  }
};

export const GetDomainBasicInfo = async (domainId: string) => {
  try {
    const accessToken = await validateOrGetToken("");
    const domain = await mgSdk.domains.Domain(domainId, accessToken);
    return {
      id: domain.id,
      name: domain.name,
      alias: domain.alias,
    } as DomainBasicInfo;
  } catch (_error) {
    return domainId;
  }
};

export const UpdateDomain = async (domain: Domain) => {
  try {
    const accessToken = await validateOrGetToken("");
    const updated = await mgSdk.domains.UpdateDomain(domain, accessToken);
    return {
      data: updated.name as string,
      error: null,
    };
  } catch (err: unknown) {
    const knownError = err as HttpError;
    return {
      data: null,
      error: knownError.error || knownError.message || knownError.toString(),
    };
  } finally {
    revalidatePath("/domain/info");
  }
};

export const EnableDomain = async (id: string) => {
  try {
    const accessToken = await validateOrGetToken("");
    await mgSdk.domains.EnableDomain(id, accessToken);
    return {
      data: "Domain enabled",
      error: null,
    };
  } catch (err: unknown) {
    const knownError = err as HttpError;
    return {
      data: null,
      error: knownError.error || knownError.message || knownError.toString(),
    };
  } finally {
    revalidatePath("/domain/info");
    revalidatePath("/platform-management");
  }
};

export const DisableDomain = async (id: string) => {
  try {
    const accessToken = await validateOrGetToken("");
    await mgSdk.domains.DisableDomain(id, accessToken);
    return {
      data: "Domain disabled",
      error: null,
    };
  } catch (err: unknown) {
    const knownError = err as HttpError;
    return {
      data: null,
      error: knownError.error || knownError.message || knownError.toString(),
    };
  } finally {
    revalidatePath("/domain/info");
    revalidatePath("/platform-management");
  }
};

export const GetDomainPermissions = async (id: string) => {
  try {
    const accessToken = await validateOrGetToken("");
    const domainPermissions = await mgSdk.domains.DomainPermissions(
      id,
      accessToken,
    );
    return {
      data: domainPermissions,
      error: null,
    };
  } catch (err: unknown) {
    const knownError = err as HttpError;
    return {
      data: null,
      error: knownError.error || knownError.message || knownError.toString(),
    };
  }
};

export const AssignMultipleUsersToDomain = async (
  userIds: string[],
  domainId: string,
  relation: Relation,
) => {
  try {
    const accessToken = await validateOrGetToken("");
    const response = await mgSdk.domains.AddUsertoDomain(
      domainId,
      userIds,
      relation,
      accessToken,
    );
    return {
      data: response.message,
      error: null,
    };
  } catch (err: unknown) {
    const knownError = err as HttpError;
    return {
      data: null,
      error: knownError.error || knownError.message || knownError.toString(),
    };
  } finally {
    revalidatePath("/domain/users");
  }
};

export const AssignUserToMultipleDomains = async (
  domainIds: string[],
  userId: string,
  relation: Relation,
) => {
  const results = await Promise.all(
    domainIds.map((domainId) =>
      AssignMultipleUsersToDomain([userId], domainId, relation),
    ),
  );
  const errors = results.filter((result) => result.error != null);
  return {
    results,
    errors,
  };
};

export const UnassignUserFromDomain = async (userId: string) => {
  try {
    const session = await getServerSession();
    const accessToken = await validateOrGetToken("");
    if (session.domain?.id && session.domain?.id !== "") {
      const response = await mgSdk.domains.RemoveUserfromDomain(
        session.domain.id,
        userId,
        accessToken,
      );
      return {
        data: response.message,
        error: null,
      };
    }
  } catch (err: unknown) {
    const knownError = err as HttpError;
    return {
      data: null,
      error: knownError.error || knownError.message || knownError.toString(),
    };
  } finally {
    revalidatePath("/domain/users");
  }
};
