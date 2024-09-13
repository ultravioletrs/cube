import SDK, { type Token, type Login } from "@absmach/magistrala-sdk";
import camelcaseKeysDeep from "camelcase-keys-deep";
import { decodeJwt } from "jose";

import { sdkConf } from "@/lib/magistrala";
import type {
  Domain as AuthDomain,
  User as AuthUser,
  UserInfo as AuthUserInfo,
} from "@/types/auth";
import {
  decodeSessionToken,
  encodeSessionToken,
  getTokenExpiry,
} from "./nextauth";

export function LoginAndGetUser(credential: Login): Promise<AuthUser> {
  return new Promise((resolve, reject) => {
    const mgSdk = new SDK(sdkConf);

    mgSdk.users
      .CreateToken(credential)
      .then((token) => {
        mgSdk.users
          .UserProfile(token.access_token)
          .then((mgUserProfile) => {
            const user = {
              id: mgUserProfile.id,
              username: mgUserProfile.name,
              name: mgUserProfile.name,
              email: mgUserProfile.credentials?.identity,
              role: mgUserProfile.role,
            } as AuthUserInfo;

            const accessDecoded = decodeJwt(token?.access_token);

            if (accessDecoded.domain && accessDecoded.domain !== "") {
              mgSdk.domains
                .Domain(accessDecoded.domain as string, token.access_token)
                .then((mgDomain) => {
                  const domain = {
                    id: mgDomain.id,
                    name: mgDomain.name,
                    alias: mgDomain.alias,
                  } as AuthDomain;
                  resolve({
                    user,
                    domain,
                    ...camelcaseKeysDeep(token),
                  } as AuthUser);
                })
                .catch((error) => {
                  reject(error);
                });
            } else {
              resolve({
                user,
                ...camelcaseKeysDeep(token),
              } as AuthUser);
            }
          })
          .catch((error) => {
            reject(error);
          });
      })
      .catch((error) => {
        reject(error);
      });
  });
}

export const RefreshToken = (
  refreshToken: string,
  domainId = "",
): Promise<Token> => {
  return new Promise((resolve, reject) => {
    try {
      const mgSdk = new SDK(sdkConf);
      resolve(
        // biome-ignore lint/style/useNamingConvention: This is from an external library
        mgSdk.users.RefreshToken({ domain_id: domainId }, refreshToken),
      );
    } catch (error) {
      reject(error);
    }
  });
};

export const DomainLoginSession = async (
  csrfToken: string,
  sessionToken: string,
  domainId: string,
): Promise<string | undefined> => {
  try {
    const decodedToken = await decodeSessionToken(csrfToken, sessionToken);
    if (!decodedToken) {
      return;
    }
    const domainToken = await RefreshToken(
      decodedToken.refreshToken as string,
      domainId,
    );

    const mgSdk = new SDK(sdkConf);

    const mgDomain = await mgSdk.domains.Domain(
      domainId,
      domainToken.access_token,
    );

    if (!mgDomain) {
      return;
    }

    return await encodeSessionToken({
      ...decodedToken,
      domain: {
        id: mgDomain.id,
        name: mgDomain.name,
        alias: mgDomain.alias,
      } as AuthDomain,
      accessToken: domainToken.access_token,
      refreshToken: domainToken.refresh_token,
      refreshTokenExpiry: getTokenExpiry(domainToken.refresh_token),
      accessTokenExpiry: getTokenExpiry(domainToken.access_token),
    });
  } catch (error) {
    // biome-ignore lint/complexity/noUselessCatch: TODO: To be fixed with a toast notification
    throw error;
  }
};
