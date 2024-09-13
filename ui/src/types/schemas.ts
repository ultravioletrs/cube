import { z } from "zod";

export const Credentials = z.object({
  identity: z.string().email(),
  secret: z.string(),
});
export const status = z.enum(["enabled", "disabled"]);

export const userSchema = z.object({
  id: z.string(),
  name: z.string(),
  credentials: Credentials,
  tags: z.array(z.string()),
  metadata: z.object({}),
  createdAt: z.date(),
  updatedAt: z.date(),
  status: status,
  role: z.string(),
  permissions: z.string().array(),
});

export const invitationSchema = z.object({
  invitedBy: z.object({}),
  // biome-ignore lint/style/useNamingConvention: This is from an external library
  user_id: z.object({}),
  // biome-ignore lint/style/useNamingConvention: This is from an external library
  domain_id: z.object({}),
  relation: z.string(),
  createdAt: z.date(),
  updatedAt: z.date(),
  confirmedAt: z.date().nullable(),
});
