import Breadcrumbs, { type BreadcrumbProps } from "@/components/breadcrumbs";
import { DisplayStatusWithIcon } from "@/components/entities/status-display-with-icon";
import ErrorComponent from "@/components/error-component";
import { Badge } from "@/components/ui/badge";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { getServerSession } from "@/lib/nextauth";
import { UserProfile } from "@/lib/users";
import ProfileForm from "./_components/profile-form";
import ProfileSettings from "./_components/profile-settings";
import SecurityForm from "./_components/security-form";

export default async function Profile() {
  const session = await getServerSession();
  const user = await UserProfile(session.accessToken);
  let breadcrumb: BreadcrumbProps[] = [];
  if (session.domain) {
    breadcrumb = [
      { label: "HomePage", href: "/domain/info" },
      {
        label: "Profile",
        href: "/profile",
        active: true,
      },
    ];
  } else {
    breadcrumb = [
      { label: "Domain Login", href: "/" },
      {
        label: "Profile",
        href: "/profile",
        active: true,
      },
    ];
  }
  return (
    <div className="container md:w-full mx-auto mt-4 pt-14 pb-4 md:pb-10">
      {user.error !== null ? (
        <div className="mt-40">
          <ErrorComponent link="/" linkText="Go Back to Domain Login" />
        </div>
      ) : (
        <>
          <Breadcrumbs breadcrumbs={breadcrumb} />
          <h3 className="text-3xl font-sans mb-4 font-semibold">
            Profile Page
          </h3>
          <p className="text-gray-500 mb-4">
            Welcome {session.user?.name ?? ""}. Manage your account settings
            here.
          </p>
          <div className="flex flex-col md:flex-row justify-between mb-4">
            <div className="flex justify-between items-center flex-row gap-4">
              <span>Your Platform Role:</span>
              <Badge
                variant="outline"
                className="w-20 h-6 text-base items-center justify-center"
              >
                {user.data?.role ?? "user"}
              </Badge>
            </div>
            <div className="flex flex-row items-center justify-between gap-4">
              <span>Your Current Status:</span>
              <DisplayStatusWithIcon status={user.data?.status as string} />
            </div>
          </div>

          <Tabs defaultValue="account" className="w-full">
            <TabsList className="grid w-[400px] grid-cols-3">
              <TabsTrigger value="account">Account</TabsTrigger>
              <TabsTrigger value="password">Password</TabsTrigger>
              <TabsTrigger value="settings">Settings</TabsTrigger>
            </TabsList>
            <TabsContent value="account">
              <ProfileForm serverSession={session} />
            </TabsContent>
            <TabsContent value="password">
              <SecurityForm />
            </TabsContent>
            <TabsContent value="settings">
              <ProfileSettings user={user.data} />
            </TabsContent>
          </Tabs>
        </>
      )}
    </div>
  );
}
