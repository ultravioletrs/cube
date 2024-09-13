import ErrorComponent from "@/components/error-component";
import { GetUsers } from "@/lib/users";
import type { Status, UsersPage } from "@absmach/magistrala-sdk";
import { BulkCreate } from "./_components/bulk-create";
import { CreateUserForm } from "./_components/create-form";
import { UsersTable } from "./_components/users-table";

const Users = async ({
  searchParams,
}: {
  searchParams?: {
    name?: string;
    id?: string;
    identity?: string;
    page?: string;
    limit?: string;
    status?: Status;
    tag?: string;
  };
}) => {
  const page = Number(searchParams?.page) || 1;
  const limit = Number(searchParams?.limit) || 10;
  const name = searchParams?.name || "";
  const searchId = searchParams?.id || "";
  const identity = searchParams?.identity || "";
  const tag = searchParams?.tag || "";
  const status = searchParams?.status || "enabled";
  const response = await GetUsers({
    queryParams: {
      offset: (page > 0 ? page - 1 : 0) * limit,
      limit,
      name: name.trim(),
      id: searchId.trim(),
      identity: identity.trim(),
      tag: tag.trim(),
      status: status,
    },
  });

  return (
    <>
      {response.error !== null ? (
        <div className="mt-40">
          <ErrorComponent link="/" linkText="Go Back to Domain Login" />
        </div>
      ) : (
        <>
          <div className="flex item-center justify-end gap-2">
            <CreateUserForm />
            <BulkCreate />
          </div>
          <UsersTable
            usersPage={response.data as UsersPage}
            page={page}
            limit={limit}
          />
        </>
      )}
    </>
  );
};

export default Users;
