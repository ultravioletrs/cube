import HomeTopbar from "@/components/layout/home-topbar";
import { Navbars } from "@/components/layout/navbars";
import { GetDomainPermissions, GetDomains } from "@/lib/domains";
import { GetInvitations } from "@/lib/invitations";
import { getServerSession } from "@/lib/nextauth";
import { UserProfile } from "@/lib/users";
import type { Permissions, User } from "@absmach/magistrala-sdk";
import type { ReactNode } from "react";

export default async function HomeRootLayout({
  children,
}: {
  children: ReactNode;
}) {
  const session = await getServerSession();
  if (session.domain) {
    const domainPermissions = await GetDomainPermissions(
      session.domain.id as string,
    );
    const domainResponse = await GetDomains({ queryParams: { limit: 5 } });
    const invitationResponse = await GetInvitations({
      queryParams: {
        limit: 20,
        state: "pending",
        // biome-ignore lint/style/useNamingConvention: This is from an external library
        user_id: session?.user?.id,
      },
    });
    const userResponse = await UserProfile(session.accessToken);
    const user = userResponse.data as User;
    return (
      <>
        <div className="flex flex-row w-[100%]">
          <Navbars
            domainPermissions={domainPermissions.data as Permissions}
            domainResponse={domainResponse}
            invitationResponse={invitationResponse}
            session={session}
            user={user}
          />
          {children}
        </div>
      </>
    );
  }

  return (
    <>
      <HomeTopbar />
      {children}
    </>
  );
}
