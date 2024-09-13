"use server";
import { DomainLoginSession } from "@/lib/auth";
import { GetDomainUsers } from "@/lib/domains";
import { GetDomains } from "@/lib/domains";
import { getServerSession } from "@/lib/nextauth";
import { UserProfile } from "@/lib/users";
import { GetUsers } from "@/lib/users";
import type { Domain as AuthDomain, UserInfo } from "@/types/auth";
import { EntityType } from "@/types/entities";
import type { HttpError } from "@/types/errors";
import type { PageMetadata, User } from "@absmach/magistrala-sdk";
import type { Domain } from "@absmach/magistrala-sdk";
import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { GetDomainInfo } from "./domains";
import {
  decodeSessionToken,
  encodeSessionToken,
  getTokenExpiry,
} from "./nextauth";

export async function DomainLogin(domainId: string) {
  const cookiesStore = cookies();

  const session = await getServerSession();
  if (session.error) {
    cookiesStore.delete("next-auth.session-token");
    redirect(`/login?error=${session.error}`);
  }
  if (session.domain?.id && session.domain.id === domainId) {
    redirect("/domain/info");
  }

  const sessionTokenCookie = cookiesStore.get("next-auth.session-token");
  const csrfTokenCookie = cookiesStore.get("next-auth.csrf-token");

  if (!csrfTokenCookie?.value || !sessionTokenCookie?.value) {
    return;
  }

  const domainSession = await DomainLoginSession(
    csrfTokenCookie.value,
    sessionTokenCookie.value,
    domainId,
  );

  if (!domainSession) {
    return;
  }
  cookiesStore.set("next-auth.session-token", domainSession);

  redirect("/domain/info");
}

export const UpdateServerSession = async (): Promise<string | undefined> => {
  try {
    const cookiesStore = cookies();
    const csrfTokenCookie = cookiesStore.get("next-auth.csrf-token");
    const sessionTokenCookie = cookiesStore.get("next-auth.session-token");
    const decodedToken = await decodeSessionToken(
      csrfTokenCookie?.value as string,
      sessionTokenCookie?.value as string,
    );
    if (!decodedToken) {
      return;
    }

    const user = (await UserProfile(
      decodedToken.accessToken as string,
    )) as User;
    const domain: Domain = (await GetDomainInfo()) as Domain;

    const updatedSession = await encodeSessionToken({
      ...decodedToken,
      user: {
        id: user.id as string,
        username: user.name as string,
        name: user.name as string,
        email: user.credentials?.identity as string,
      } as UserInfo,
      domain: {
        id: domain.id,
        name: domain.name,
        alias: domain.alias,
      } as AuthDomain,
      accessToken: decodedToken.accessToken,
      refreshToken: decodedToken.refreshToken,
      refreshTokenExpiry: getTokenExpiry(decodedToken.refreshToken as string),
      accessTokenExpiry: getTokenExpiry(decodedToken.accessToken as string),
    });

    cookiesStore.set("next-auth.session-token", updatedSession);
  } catch (error) {
    // biome-ignore lint/complexity/noUselessCatch: TODO: To be fixed with a toast notification
    throw error;
  }
};

export interface EntityFetchData {
  total: number;
  data: User[];
  limit: number;
  error: string | null;
}

export const FetchData = async (
  entity: EntityType,
  queryParams: PageMetadata,
  id?: string,
  isEntityCard?: boolean,
): Promise<EntityFetchData> => {
  try {
    switch (entity) {
      case EntityType.User: {
        const usersPage = await GetUsers({ queryParams });
        if (usersPage.error !== null) {
          const knownError = usersPage.error as unknown as HttpError;
          return {
            total: 0,
            data: [],
            limit: 0,
            error:
              knownError.error || knownError.message || knownError.toString(),
          };
        }
        return {
          total: usersPage.data.total,
          data: usersPage.data.users,
          limit: usersPage.data.limit,
          error: null,
        };
      }
      case EntityType.Member: {
        if (!isEntityCard) {
          queryParams.permission = "member";
        }
        const membersPage = await GetDomainUsers({ queryParams });
        if (membersPage.error !== null) {
          const knownError = membersPage.error as unknown as HttpError;
          return {
            total: 0,
            data: [],
            limit: 0,
            error:
              knownError.error || knownError.message || knownError.toString(),
          };
        }
        return {
          total: membersPage.data.total,
          data: membersPage.data.users,
          limit: membersPage.data.limit,
          error: null,
        };
      }
      case EntityType.Domain: {
        const domains = await GetDomains({ id, queryParams });
        if (domains.error !== null) {
          const knownError = domains.error as unknown as HttpError;
          return {
            total: 0,
            data: [],
            limit: 0,
            error:
              knownError.error || knownError.message || knownError.toString(),
          };
        }
        return {
          total: domains.data.total,
          data: domains.data.domains,
          limit: domains.data.limit,
          error: null,
        };
      }
      default:
        return {
          total: 0,
          data: [],
          limit: 0,
          error: "Invalid entity type",
        };
    }
  } catch (err: unknown) {
    const knownError = err as HttpError;
    return {
      total: 0,
      data: [],
      limit: 0,
      error: knownError.error || knownError.message || knownError.toString(),
    };
  }
};
