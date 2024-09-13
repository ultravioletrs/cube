export interface UserInfo {
  id?: string;
  username?: string;
  name?: string;
  email?: string;
  role?: UserRole;
  image?: string;
}

export interface Domain {
  id?: string;
  name?: string;
  alias?: string;
}

export interface AccessToken {
  accessToken: string;
  accessTokenExpiry: number;
}
export interface RefreshToken {
  refreshToken: string;
  refreshTokenExpiry: number;
}
export interface Tokens extends AccessToken, RefreshToken {}

export interface Session extends AccessToken {
  user: UserInfo;
  domain?: Domain;
  error?: string;
  expires: string;
}

export interface User extends Tokens {
  user: UserInfo;
  domain?: Domain;
  error?: string;
}

export enum UserRole {
  Admin = "admin",
  User = "user",
}
