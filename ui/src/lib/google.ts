"use server";
interface OauthConfig {
  clientId: string;
  clientSecret: string;
  redirectUrl: string;
  state: string;
}

const scopes: string = [
  "https://www.googleapis.com/auth/userinfo.email",
  "https://www.googleapis.com/auth/userinfo.profile",
].join(" ");
const authUrl = "https://accounts.google.com/o/oauth2/auth";
const responseType = "code";
const accessType = "offline";
const prompt = "consent";
const config: OauthConfig = {
  clientId: process.env.MG_GOOGLE_CLIENT_ID || "",
  clientSecret: process.env.MG_GOOGLE_CLIENT_SECRET || "",
  redirectUrl: process.env.MG_GOOGLE_REDIRECT_URL || "",
  state: process.env.MG_GOOGLE_STATE || "",
};

export const GenerateGoogleUrl = () => {
  const url = new URL(authUrl);
  const parameters = new URLSearchParams({
    // biome-ignore lint/style/useNamingConvention: This is from an external library
    client_id: config.clientId,
    scope: scopes,
    // biome-ignore lint/style/useNamingConvention: This is from an external library
    redirect_uri: config.redirectUrl,
    // biome-ignore lint/style/useNamingConvention: This is from an external library
    response_type: responseType,
    // biome-ignore lint/style/useNamingConvention: This is from an external library
    access_type: accessType,
    prompt: prompt,
    state: config.state,
  });

  url.search = parameters.toString();
  return url.toString();
};

export const IsGoogleEnabled = (): boolean => {
  return config.clientId !== "" && config.clientSecret !== "";
};
