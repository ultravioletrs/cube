"use server";
import type { HttpError } from "@/types/errors";
import type { PageMetadata, User } from "@absmach/magistrala-sdk";
import { revalidatePath } from "next/cache";
import { redirect } from "next/navigation";
import { type RequestOptions, validateOrGetToken } from "./magistrala";
import { mgSdk } from "./magistrala";

export const GetUsers = async ({ token = "", queryParams }: RequestOptions) => {
  try {
    const accessToken = await validateOrGetToken(token);
    const users = await mgSdk.users.Users(queryParams, accessToken);
    return {
      data: users,
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

export const CreateUser = async (user: User) => {
  try {
    const accessToken = await validateOrGetToken("");
    const created = await mgSdk.users.Create(user, accessToken);
    return {
      data: created.name as string,
      error: null,
    };
  } catch (err: unknown) {
    const knownError = err as HttpError;
    return {
      data: null,
      error: knownError.error || knownError.message || knownError.toString(),
    };
  } finally {
    revalidatePath("/platform-management/users");
  }
};

export const ViewUser = async (id: string) => {
  try {
    const accessToken = await validateOrGetToken("");
    const user = await mgSdk.users.User(id, accessToken);
    return {
      data: user,
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

export const DisableUser = async (id: string) => {
  const user: User = {
    id: id,
    status: "disabled",
  };
  try {
    const accessToken = await validateOrGetToken("");
    await mgSdk.users.Disable(user, accessToken);
    return {
      data: "User disabled",
      error: null,
    };
  } catch (err: unknown) {
    const knownError = err as HttpError;
    return {
      data: null,
      error: knownError.error || knownError.message || knownError.toString(),
    };
  } finally {
    revalidatePath(`/platform-management/users/${user.id}`);
  }
};

export const EnableUser = async (id: string) => {
  const user: User = {
    id: id,
  };
  try {
    const accessToken = await validateOrGetToken("");
    await mgSdk.users.Enable(user, accessToken);
    return {
      data: "User enabled",
      error: null,
    };
  } catch (err: unknown) {
    const knownError = err as HttpError;
    return {
      data: null,
      error: knownError.error || knownError.message || knownError.toString(),
    };
  } finally {
    revalidatePath(`/platform-management/users/${user.id}`);
  }
};

export const UpdateUserRole = async (formData: FormData) => {
  const user: User = {
    id: formData.get("entityId") as string,
    role: formData.get("role") as string,
  };
  try {
    const accessToken = await validateOrGetToken("");
    const updated = await mgSdk.users.UpdateUserRole(user, accessToken);
    return {
      data: updated.name as string,
      error: null,
    };
  } catch (err: unknown) {
    const knownError = err as HttpError;
    return {
      data: null,
      error: knownError.error || knownError.message || knownError.toString(),
    };
  } finally {
    revalidatePath(`/platform-management/users/${user.id}`);
  }
};

export const UpdateUser = async (user: User) => {
  try {
    const accessToken = await validateOrGetToken("");
    const updated = await mgSdk.users.Update(user, accessToken);
    return {
      data: updated.name as string,
      error: null,
    };
  } catch (err: unknown) {
    const knownError = err as HttpError;
    return {
      data: null,
      error: knownError.error || knownError.message || knownError.toString(),
    };
  } finally {
    revalidatePath(`/platform-management/users/${user.id}`);
  }
};

export const UpdateUserIdentity = async (user: User) => {
  try {
    const accessToken = await validateOrGetToken("");
    const updated = await mgSdk.users.UpdateUserIdentity(user, accessToken);
    return {
      data: updated.name as string,
      error: null,
    };
  } catch (err: unknown) {
    const knownError = err as HttpError;
    return {
      data: null,
      error: knownError.error || knownError.message || knownError.toString(),
    };
  } finally {
    revalidatePath(`/platform-management/users/${user.id}`);
  }
};

export const UpdateUserTags = async (user: User) => {
  try {
    const accessToken = await validateOrGetToken("");
    const updated = await mgSdk.users.UpdateUserTags(user, accessToken);
    return {
      data: updated.name as string,
      error: null,
    };
  } catch (err: unknown) {
    const knownError = err as HttpError;
    return {
      data: null,
      error: knownError.error || knownError.message || knownError.toString(),
    };
  } finally {
    revalidatePath(`/platform-management/users/${user.id}`);
  }
};

export const ResetPasswordRequest = async (email: string) => {
  try {
    await mgSdk.users.ResetPasswordRequest(email);
    return {
      data: "Password reset link sent successfully. Check your email.",
      error: null,
    };
  } catch (err: unknown) {
    const knownError = err as HttpError;
    return {
      error:
        knownError.error ||
        knownError.message ||
        "Failed to send password reset link.",
    };
  } finally {
    revalidatePath("/login");
  }
};

export const ResetPassword = async (
  password: string,
  confirmPassword: string,
  token: string,
) => {
  try {
    await mgSdk.users.ResetPassword(password, confirmPassword, token);
  } catch (err: unknown) {
    const knownError = err as HttpError;
    return {
      error:
        knownError.error || knownError.message || "Failed to reset password",
    };
  }
  redirect("/login");
};

export const UpdateUserPassword = async (
  oldSecret: string,
  newSecret: string,
) => {
  try {
    const accessToken = await validateOrGetToken("");
    await mgSdk.users.UpdateUserPassword(oldSecret, newSecret, accessToken);
    return {
      data: "Password Updated Successfully",
      error: null,
    };
  } catch (err: unknown) {
    const knownError = err as HttpError;
    return {
      error: knownError.message || knownError.toString(),
    };
  } finally {
    revalidatePath("/profile");
  }
};

export const UserProfile = async (token?: string) => {
  try {
    const accessToken = await validateOrGetToken(token as string);
    const user = await mgSdk.users.UserProfile(accessToken);
    return {
      data: user,
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

export const DeleteUser = async (id: string) => {
  try {
    const accessToken = await validateOrGetToken("");
    const response = await mgSdk.users.DeleteUser(id, accessToken);
    return {
      data: response.message,
      error: null,
    };
  } catch (err: unknown) {
    const knownError = err as HttpError;
    return {
      data: null,
      error: knownError.error || knownError.message || knownError.toString(),
    };
  } finally {
    revalidatePath("/platform-management/users");
  }
};

export const SearchUsers = async (queryParams: PageMetadata) => {
  try {
    const accessToken = await validateOrGetToken("");
    const users = await mgSdk.users.SearchUsers(queryParams, accessToken);
    return {
      data: users,
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
