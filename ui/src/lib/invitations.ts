"use server";
import type { HttpError } from "@/types/errors";
import type {
  DomainBasicInfo,
  Invitation,
  InvitationsPage,
  Relation,
  UserBasicInfo,
} from "@absmach/magistrala-sdk";
import { revalidatePath } from "next/cache";
import { GetDomainBasicInfo, GetUserBasicInfo } from "./domains";
import { mgSdk } from "./magistrala";
import { type RequestOptions, validateOrGetToken } from "./magistrala";

export const SendInvitation = async (invitation: Invitation) => {
  try {
    const accessToken = await validateOrGetToken("");
    await mgSdk.invitations.SendInvitation(invitation, accessToken);
    return {
      data: "Invitation sent successfully",
      error: null,
    };
  } catch (err: unknown) {
    const knownError = err as HttpError;
    return {
      data: null,
      error: knownError.message || knownError.toString(),
    };
  } finally {
    revalidatePath("/invitations");
  }
};

export const InviteMultipleUsersToDomain = async (
  userIds: string[],
  domainId: string,
  relation: Relation,
) => {
  const results = await Promise.all(
    userIds.map((userId) => {
      const invitation: Invitation = {
        // biome-ignore lint/style/useNamingConvention: This is from an external library
        domain_id: domainId,
        // biome-ignore lint/style/useNamingConvention: This is from an external library
        user_id: userId,
        relation: relation,
      };
      return SendInvitation(invitation);
    }),
  );
  const errors = results.filter((result) => result.error != null);
  return {
    results,
    errors,
  };
};

export const InviteUserToMultipleDomains = async (
  domainIds: string[],
  userId: string,
  relation: Relation,
) => {
  const results = await Promise.all(
    domainIds.map((domainId) => {
      const invitation: Invitation = {
        // biome-ignore lint/style/useNamingConvention: This is from an external library
        domain_id: domainId,
        // biome-ignore lint/style/useNamingConvention: This is from an external library
        user_id: userId,
        relation: relation,
      };
      return SendInvitation(invitation);
    }),
  );
  const errors = results.filter((result) => result.error != null);
  return {
    results,
    errors,
  };
};

export const GetInvitations = async ({
  token = "",
  queryParams,
}: RequestOptions) => {
  try {
    const accessToken = await validateOrGetToken(token);
    const invitationsPage = await mgSdk.invitations.Invitations(
      queryParams,
      accessToken,
    );

    const invitations = await processInvitations(
      invitationsPage.invitations,
      accessToken,
    );
    return {
      data: {
        total: invitationsPage.total,
        offset: invitationsPage.offset,
        limit: invitationsPage.limit,
        invitations,
      } as InvitationsPage,
      error: null,
    };
  } catch (err: unknown) {
    const knownError = err as HttpError;
    return {
      data: null,
      error: knownError.error || knownError.message || knownError.toString(),
    };
  }
};

export const GetInvitation = async (userId: string, domainId: string) => {
  try {
    const accessToken = await validateOrGetToken("");
    const invitation = mgSdk.invitations.Invitation(
      userId,
      domainId,
      accessToken,
    );

    return invitation;
  } catch (err: unknown) {
    const knownError = err as HttpError;
    throw new Error(
      knownError.error || knownError.message || knownError.toString(),
    );
  }
};

export const AcceptInvitation = async (domainId: string) => {
  try {
    const accessToken = await validateOrGetToken("");
    await mgSdk.invitations.AcceptInvitation(domainId, accessToken);
    return {
      data: "Invitation Accepted Successfully",
      error: null,
    };
  } catch (err: unknown) {
    const knownError = err as HttpError;
    return {
      data: null,
      error: knownError.message || knownError.toString(),
    };
  } finally {
    revalidatePath("/domain");
  }
};

export const DeleteInvitation = async (domainId: string, userId: string) => {
  try {
    const accessToken = await validateOrGetToken("");
    await mgSdk.invitations.DeleteInvitation(userId, domainId, accessToken);
    return {
      data: "Inviatation Deleted Successfully",
      error: null,
    };
  } catch (err: unknown) {
    const knownError = err as HttpError;
    return {
      data: null,
      error: knownError.error || knownError.message || knownError.toString(),
    };
  } finally {
    revalidatePath("/domain");
  }
};

async function processInvitations(
  invitations: Invitation[],
  token: string,
): Promise<Invitation[]> {
  const processedInvitations: Invitation[] = [];
  if (invitations && invitations.length > 0) {
    for (const invitation of invitations) {
      try {
        const invitedBy: UserBasicInfo | string =
          typeof invitation.invited_by === "string"
            ? invitation.invited_by === ""
              ? invitation.invited_by
              : await GetUserBasicInfo(invitation.invited_by, token)
            : (invitation.invited_by as UserBasicInfo);

        const user: UserBasicInfo | string =
          typeof invitation.user_id === "string"
            ? invitation.user_id === ""
              ? invitation.user_id
              : await GetUserBasicInfo(invitation.user_id, token)
            : (invitation.user_id as UserBasicInfo);

        const domainId: DomainBasicInfo | string =
          typeof invitation.domain_id === "string"
            ? invitation.domain_id === ""
              ? invitation.domain_id
              : await GetDomainBasicInfo(invitation.domain_id)
            : (invitation.domain_id as DomainBasicInfo);

        const processedInvitation: Invitation = {
          ...invitation,
          // biome-ignore lint/style/useNamingConvention: This is from an external library
          invited_by: invitedBy,
          // biome-ignore lint/style/useNamingConvention: This is from an external library
          user_id: user,
          // biome-ignore lint/style/useNamingConvention: This is from an external library
          domain_id: domainId,
        };

        processedInvitations.push(processedInvitation);
      } catch (_error) {
        processedInvitations.push(invitation);
      }
    }
    return processedInvitations;
  }
  return invitations;
}
