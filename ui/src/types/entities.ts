import type { GroupRelation, Relation } from "@absmach/magistrala-sdk";

export enum EntityType {
  User = "user",
  Domain = "domain",
  Member = "member",
}

export interface Metadata {
  // biome-ignore lint/suspicious/noExplicitAny: This is a valid use case for any
  [key: string]: any;
}

export const DomainRelations: Relation[] = [
  "administrator",
  "editor",
  "contributor",
  "member",
  "guest",
];

export const GroupRelations: GroupRelation[] = [
  "administrator",
  "editor",
  "contributor",
  "guest",
];

export enum Status {
  Enabled = "enabled",
  Disabled = "disabled",
  All = "all",
}

export enum Themes {
  MidnightSky = "midnightsky",
  TealTide = "tealtide",
  Default = "default",
}

export const ThemeArray: string[] = Object.values(Themes);
