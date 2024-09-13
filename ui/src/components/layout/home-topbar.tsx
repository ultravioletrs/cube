import { ThemeToggle } from "@/components/layout/theme-toggle";
import { UserNav } from "@/components/layout/user-nav";
import { GetInvitations } from "@/lib/invitations";
import { getServerSession } from "@/lib/nextauth";
import { UserProfile } from "@/lib/users";
import { UserRole } from "@/types/auth";
import { Themes } from "@/types/entities";
import type { InvitationsPage, User } from "@absmach/magistrala-sdk";
import { getImageProps } from "next/image";
import Link from "next/link";
import { ViewUserInvitations } from "../invitations/viewinvitations";

export default async function HomeTopbar() {
  const common = { alt: "Abstract Machines Logo", sizes: "100vw" };
  const session = await getServerSession();
  const invitationResponse = await GetInvitations({
    queryParams: {
      limit: 20,
      state: "pending",
      // biome-ignore lint/style/useNamingConvention: This is from an external library
      user_id: session?.user?.id,
    },
  });
  const userResponse = await UserProfile(session.accessToken);
  const theme = userResponse.data?.metadata?.ui?.theme;
  const logoSrc =
    theme === Themes.MidnightSky
      ? "/abstract-machines_logo_landscape-white.svg"
      : "/abstract-machines_logo_landscape-black.svg";
  const {
    props: { srcSet: mobile },
  } = getImageProps({
    ...common,
    width: 64,
    height: 64,
    quality: 70,
    src: logoSrc,
  });

  const {
    props: { srcSet: desktop, ...rest },
  } = getImageProps({
    ...common,
    width: 200,
    height: 24,
    quality: 80,
    src: logoSrc,
  });

  return (
    <div className="fixed top-0 left-0 right-0 supports-backdrop-blur:bg-background/60 border-b bg-card backdrop-blur z-20">
      <nav className="h-14 flex items-center justify-between px-4">
        <div className="flex items-center justify-start space-x-2">
          <div>
            <Link
              href={"https://github.com/absmach/magistrala"}
              target="_blank"
            >
              <picture>
                <source media="(min-width: 999px)" srcSet={mobile} />
                <source media="(min-width: 1000px)" srcSet={desktop} />
                <img {...rest} alt="Abstract Machines Logo" />
              </picture>
            </Link>
          </div>
        </div>

        <div className="flex items-center gap-4">
          <ViewUserInvitations
            invitations={invitationResponse.data as InvitationsPage}
            error={invitationResponse.error}
          />
          <UserNav
            role={
              session.user.role
                ? (session.user.role as UserRole)
                : UserRole.User
            }
          />
          <ThemeToggle user={userResponse.data as User} />
        </div>
      </nav>
    </div>
  );
}
