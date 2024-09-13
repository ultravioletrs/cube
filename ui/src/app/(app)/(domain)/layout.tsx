import { DomainTopbar } from "@/components/layout/domain-topbar";
import { Navbars } from "@/components/layout/navbars";
import { ScrollArea } from "@/components/ui/scroll-area";
import { GetDomainPermissions } from "@/lib/domains";
import { GetDomains } from "@/lib/domains";
import { GetInvitations } from "@/lib/invitations";
import { getServerSession } from "@/lib/nextauth";
import { UserProfile } from "@/lib/users";
import type { Permissions } from "@absmach/magistrala-sdk";
import type { User } from "@absmach/magistrala-sdk";
import type { ReactNode } from "react";

export default async function DomainRootLayout({
  children,
}: {
  children: ReactNode;
}) {
  const session = await getServerSession();
  const domainPermissions = await GetDomainPermissions(
    session.domain?.id as string,
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

  return (
    <>
      <DomainTopbar
        domainPermissions={domainPermissions.data as Permissions}
        session={session}
        domainResponse={domainResponse}
        invitationResponse={invitationResponse}
        expand={true}
        user={userResponse.data as User}
      />
      <div className="flex flex-row w-[100%]">
        <aside className="fixed top-0 z-30 hidden h-[calc(100vh-28rem)] shrink-0 md:sticky md:block">
          <ScrollArea>
            <Navbars
              domainPermissions={domainPermissions.data as Permissions}
              domainResponse={domainResponse}
              invitationResponse={invitationResponse}
              session={session}
              user={userResponse.data as User}
            />
          </ScrollArea>
        </aside>
        <div className="w-full pt-14 pb-2 mt-4">{children}</div>
      </div>
    </>
  );
}
