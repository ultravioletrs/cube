import { getServerSession } from "@/lib/nextauth";
import SDK, {
  type PageMetadata,
  type SDKConfig,
} from "@absmach/magistrala-sdk";

export const sdkConf: SDKConfig = {
  usersUrl: process.env.MG_USERS_URL || "",
  domainsUrl: process.env.MG_DOMAINS_URL || "",
  hostUrl: process.env.MG_BASE_URL || "",
  invitationsUrl: process.env.MG_INVITATIONS_URL || "",
};

export interface RequestOptions {
  token?: string;
  id?: string;
  queryParams: PageMetadata;
}

export const mgSdk = new SDK(sdkConf);

export const validateOrGetToken = async (token: string) => {
  if (token) {
    return token;
  }
  const session = await getServerSession();
  if (session && session.accessToken !== "") {
    return session.accessToken;
  }
  return "";
};
