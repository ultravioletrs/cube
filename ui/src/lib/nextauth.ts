import { createHash } from "node:crypto";
import { decodeJwt } from "jose";
import type {
  GetServerSidePropsContext,
  NextApiRequest,
  NextApiResponse,
} from "next";
import {
  type NextAuthOptions,
  getServerSession as getNextAuthServerSession,
} from "next-auth";
import { decode, encode } from "next-auth/jwt";
import CredentialsProvider from "next-auth/providers/credentials";

import type { Session as AuthSession, User as AuthUser } from "@/types/auth";

declare module "next-auth" {
  interface Session extends AuthSession {}
  interface User extends AuthUser {}
}
declare module "next-auth/jwt" {
  interface Jwt extends AuthUser {}
}
import type { HttpError } from "@/types/errors";
import type { User } from "next-auth";
import type { JWT } from "next-auth/jwt";
import NextAuth from "next-auth/next";
import { LoginAndGetUser, RefreshToken } from "./auth";

const authOptions: NextAuthOptions = {
  providers: [
    CredentialsProvider({
      credentials: {
        domain: {
          label: "domain",
          type: "text",
          placeholder: "domain id",
        },
        email: {
          label: "email",
          type: "email",
          placeholder: "example@example.com",
        },
        password: {
          label: "password",
          type: "password",
          placeholder: "password",
        },
      },
      async authorize(credentials) {
        try {
          const user = await LoginAndGetUser({
            identity: credentials?.email || "",
            secret: credentials?.password || "",
            // biome-ignore lint/style/useNamingConvention: This is from external SDK
            domain_id: credentials?.domain || "",
          });
          return user as User;
        } catch (err: unknown) {
          const knownError = err as HttpError;
          throw new Error(knownError.error);
        }
      },
    }),
  ],

  session: {
    maxAge: 1 * 24 * 60 * 60, // 1 day
    updateAge: 2 * 60 * 60, // 2 hours
  },

  callbacks: {
    // biome-ignore lint/suspicious/useAwait: This is from an external library
    async jwt({ token, user, session, account, trigger }) {
      if (trigger === "signIn") {
        if (user && account) {
          return {
            ...token,
            ...user,
            refreshTokenExpiry: getTokenExpiry(user.refreshToken),
            accessTokenExpiry: getTokenExpiry(user.accessToken),
          };
        }
        return { ...token, error: "internal error" };
      }

      if (
        !token.refreshTokenExpiry ||
        Date.now().toString() >= token.refreshTokenExpiry
      ) {
        return { ...token, error: "session expired" };
      }
      if (
        !token.accessTokenExpiry ||
        Date.now().toString() >= token.accessTokenExpiry
      ) {
        return refreshAccessToken(token);
      }
      if (trigger === "update" && session) {
        token = { ...token, user: session };
        return token;
      }
      return token;
    },

    // TODO refresh like this https://github.com/nextauthjs/next-auth/issues/7913#issuecomment-1937009383
    // TODO token https://github.com/nextauthjs/next-auth/issues/7913#issuecomment-1942534614
    // @ts-ignore
    // biome-ignore lint/suspicious/useAwait: This is from an external library
    async session({ session, token }) {
      // Send properties to the client, like an access_token from a provider.
      return {
        ...session,
        user: token.user,
        domain: token.domain,
        accessToken: token?.accessToken,
        accessTokenExpiry: token?.accessTokenExpiry,
        error: token.error,
      };
    },
  },

  pages: {
    signIn: "/login",
    error: "/login",
  },
  debug: true,
};

// Reference  https://github.com/nextauthjs/next-auth/issues/7913#issuecomment-1873953645
async function getServerSession(
  ...args:
    | [GetServerSidePropsContext["req"], GetServerSidePropsContext["res"]]
    | [NextApiRequest, NextApiResponse]
    | []
) {
  const nextSession = await getNextAuthServerSession(...args, authOptions);
  return nextSession as AuthSession;
}

async function decodeSessionToken(csrfToken: string, sessionToken: string) {
  const csrf = await verifyCsrfToken(csrfToken);
  if (csrf?.verified) {
    const decoded = await decode({
      token: sessionToken,
      secret: process.env?.NEXTAUTH_SECRET || "",
    });
    return decoded;
  }
  return null;
}

async function encodeSessionToken(sessionToken: JWT): Promise<string> {
  return await encode({
    secret: process.env.NEXTAUTH_SECRET || "",
    token: sessionToken,
    maxAge: 1 * 24 * 60 * 60, // 1 day
  });
}

function verifyCsrfToken(token: string | undefined) {
  // delimiter could be either a '|' or a '%7C'
  if (token) {
    const tokenHashDelimiter = token.indexOf("|") !== -1 ? "|" : "%7C";
    const [csrfToken, csrfTokenHash] = token.split(tokenHashDelimiter);
    const expectedCsrfTokenHash = createHash("sha256")
      .update(`${csrfToken}${process.env.NEXTAUTH_SECRET}`)
      .digest("hex");
    return {
      token: csrfToken,
      verified: csrfTokenHash === expectedCsrfTokenHash,
    };
  }
}

async function refreshAccessToken(token: JWT): Promise<JWT> {
  try {
    const resp = await RefreshToken(token.refreshToken as string);
    const accessToken = resp.access_token;
    const refreshToken = resp.refresh_token ?? token.refreshToken;
    return {
      ...token,
      accessToken,
      refreshToken,
      accessTokenExpiry: getTokenExpiry(accessToken),
      refreshTokenExpiry: getTokenExpiry(refreshToken as string),
    };
  } catch (_error) {
    // For error internal server logging purpose, need to migrate to proper JS Logger.
    return {
      ...token,
      error: "internal server error",
    };
  }
}

function getTokenExpiry(token: string, multiply = 1000): number {
  const decodedToken = decodeJwt(token);
  return decodedToken.exp ? decodedToken.exp * multiply : 0;
}

export const { auth, update } = NextAuth(authOptions);

export {
  authOptions,
  getServerSession,
  decodeSessionToken,
  encodeSessionToken,
  getTokenExpiry,
};
