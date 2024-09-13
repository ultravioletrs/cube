import { getToken } from "next-auth/jwt";
import { withAuth } from "next-auth/middleware";
import {
  type NextFetchEvent,
  type NextRequest,
  NextResponse,
} from "next/server";
import { type UserInfo, UserRole } from "./types/auth";
export default async function middleware(
  req: NextRequest,
  event: NextFetchEvent,
) {
  const token = await getToken({ req });
  const domain = token?.domain as { id: string } | { id: "" };
  const tokenError = token?.error as string | "";
  const isAuthenticated = !!token && !tokenError;
  const isDomainAuthenticated =
    isAuthenticated && !!domain && !!domain?.id && domain?.id !== "";
  const user = token?.user as UserInfo;

  if (req.nextUrl.pathname.startsWith("/login")) {
    if (isAuthenticated) {
      if (req.nextUrl.searchParams.get("callbackUrl")) {
        const callback = req.nextUrl.searchParams.get("callbackUrl") as string;
        return NextResponse.redirect(new URL(callback, req.url));
      }
      return NextResponse.redirect(new URL("/", req.url));
    }
    if (tokenError) {
      const nextUrl = req.nextUrl;
      nextUrl.searchParams.set(
        "error",
        nextUrl.searchParams.get("error")
          ? `${nextUrl.searchParams.get("error")} ${tokenError}`
          : tokenError,
      );
      const resp = NextResponse.redirect(nextUrl);
      resp.cookies.delete("next-auth.session-token");
      resp.cookies.delete("next-auth.csrf-token");
      return resp;
    }
    return;
  }

  if (req.nextUrl.pathname.startsWith("/register")) {
    if (isAuthenticated) {
      return NextResponse.redirect(new URL("/", req.url));
    }
    return;
  }

  if (req.nextUrl.pathname === "/domain") {
    if (!isDomainAuthenticated) {
      return NextResponse.redirect(new URL("/", req.url));
    }
    return;
  }

  if (req.nextUrl.pathname.startsWith("/domain/")) {
    if (!isDomainAuthenticated) {
      return NextResponse.redirect(new URL("/", req.url));
    }
    return;
  }

  if (req.nextUrl.pathname.startsWith("/platform-management")) {
    if (user?.role !== UserRole.Admin) {
      return NextResponse.redirect(new URL("/not-found", req.url));
    }
  }

  const authMiddleware = withAuth({
    pages: {
      signIn: "/login",
    },
    callbacks: {
      authorized: ({ token }) => !!token && !token?.error,
    },
  });

  // @ts-expect-error
  return authMiddleware(req, event);
}

export const config = {
  matcher: [
    "/",
    "/domains",
    "/platform-management",
    "/login",
    "/register",
    "/domain/(.*)",
    "/domain/",
  ],
};
