import { encodeSessionToken, getTokenExpiry } from "@/lib/nextauth";
import { UserProfile } from "@/lib/users";
import type { UserInfo } from "@/types/auth";
import { cookies } from "next/headers";
import { type NextRequest, NextResponse } from "next/server";

// biome-ignore lint/style/useNamingConvention: This is from an external library
export async function GET(_req: NextRequest, _resp: NextResponse) {
  const cookiesStore = cookies();
  const accessToken = cookiesStore.get("access_token")?.value;
  const refreshToken = cookiesStore.get("refresh_token")?.value;
  const baseUrl = process.env.MG_BASE_URL || "";

  const response = await UserProfile(accessToken);
  if (response.error !== null) {
    return NextResponse.redirect(`${baseUrl}/login?error=${response.error}`);
  }
  const user = response.data;
  const userSession = await encodeSessionToken({
    user: {
      id: user.id as string,
      username: user.name as string,
      name: user.name as string,
      email: user.credentials?.identity as string,
    } as UserInfo,
    accessToken,
    refreshToken,
    refreshTokenExpiry: getTokenExpiry(refreshToken as string),
    accessTokenExpiry: getTokenExpiry(accessToken as string),
  });

  cookiesStore.set("next-auth.session-token", userSession);
  return NextResponse.redirect(baseUrl);
}
