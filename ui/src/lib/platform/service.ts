// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
import SDK from '@absmach/magistrala-sdk'
import type {
  Domain,
  DomainsPage,
  PageMetadata,
  BasicPageMeta,
  Invitation,
  InvitationsPage,
  InvitationPageMeta,
  MemberRoles,
  MemberRolesPage,
  User,
  UsersPage,
  RolePage,
  Journal,
  JournalsPage,
  JournalsPageMetadata,
} from '@absmach/magistrala-sdk'

export type { Domain, Invitation, User, MemberRoles, Journal }

const sdk = new SDK({
  usersUrl: window.location.origin,
  domainsUrl: window.location.origin,
  journalUrl: window.location.origin,
})

// ── Domains ────────────────────────────────────────────────────────────────

export async function listDomains(token: string): Promise<Domain[]> {
  const page: DomainsPage = await sdk.Domains.Domains({ limit: 100, offset: 0 } as PageMetadata, token)
  return page.domains ?? []
}

export async function listUserDomains(userID: string, token: string): Promise<Domain[]> {
  const page: DomainsPage = await sdk.Domains.ListUserDomains(userID, { limit: 100, offset: 0 } as PageMetadata, token)
  return page.domains ?? []
}

export async function createDomain(name: string, token: string): Promise<Domain> {
  const route = name.trim().toLowerCase().replace(/\s+/g, '-').replace(/[^a-z0-9-]/g, '')
  return sdk.Domains.CreateDomain({ name, route } as Domain, token)
}

export async function getDomain(domainID: string, token: string): Promise<Domain> {
  return sdk.Domains.Domain(domainID, token)
}

// ── Members ────────────────────────────────────────────────────────────────

export interface MembersResult {
  members: MemberRoles[]
  total: number
}

export async function listDomainMembers(domainID: string, token: string): Promise<MembersResult> {
  const page: MemberRolesPage = await sdk.Domains.ListDomainMembers(
    domainID,
    { limit: 100, offset: 0 } as BasicPageMeta,
    token,
  )
  return { members: page.members ?? [], total: page.total ?? 0 }
}

// ── Users ──────────────────────────────────────────────────────────────────

export async function listUsers(token: string): Promise<User[]> {
  const page: UsersPage = await sdk.Users.Users({ limit: 100, offset: 0 } as PageMetadata, token)
  return page.users ?? []
}

export async function searchUsers(query: string, token: string): Promise<User[]> {
  const page: UsersPage = await sdk.Users.SearchUsers({ limit: 20, offset: 0, name: query } as PageMetadata, token)
  return page.users ?? []
}

// ── Roles ──────────────────────────────────────────────────────────────────

export interface DomainRole {
  id: string
  name: string
}

export async function listDomainRoles(domainID: string, token: string): Promise<DomainRole[]> {
  try {
    const page: RolePage = await sdk.Domains.ListDomainRoles(domainID, { limit: 50, offset: 0 } as PageMetadata, token)
    return (page.roles ?? []).map(r => ({ id: r.id ?? '', name: r.name ?? '' }))
  } catch {
    return []
  }
}

// ── Invitations ────────────────────────────────────────────────────────────

export interface InvitationsResult {
  invitations: Invitation[]
  total: number
}

export async function listDomainInvitations(domainID: string, token: string): Promise<InvitationsResult> {
  const page: InvitationsPage = await sdk.Domains.ListDomainInvitations(
    { limit: 100, offset: 0 } as InvitationPageMeta,
    domainID,
    token,
  )
  return { invitations: page.invitations ?? [], total: page.total ?? 0 }
}

export async function listUserInvitations(token: string): Promise<InvitationsResult> {
  const page: InvitationsPage = await sdk.Domains.ListUserInvitations(
    { limit: 100, offset: 0 } as PageMetadata,
    token,
  )
  return { invitations: page.invitations ?? [], total: page.total ?? 0 }
}

export async function sendInvitation(userID: string, domainID: string, roleID: string, token: string): Promise<void> {
  await sdk.Domains.SendInvitation(userID, domainID, roleID, token)
}

export async function acceptInvitation(domainID: string, token: string): Promise<void> {
  await sdk.Domains.AcceptInvitation(domainID, token)
}

export async function rejectInvitation(domainID: string, token: string): Promise<void> {
  await sdk.Domains.RejectInvitation(domainID, token)
}

export async function deleteInvitation(userID: string, domainID: string, token: string): Promise<void> {
  await sdk.Domains.DeleteInvitation(userID, domainID, token)
}

// ── Journal / Audit Logs ───────────────────────────────────────────────────

export interface JournalsResult {
  journals: Journal[]
  total: number
}

export const JOURNAL_PAGE_SIZE = 10

export async function listUserJournals(userID: string, token: string, page = 0): Promise<JournalsResult> {
  const meta: JournalsPageMetadata = { limit: JOURNAL_PAGE_SIZE, offset: page * JOURNAL_PAGE_SIZE }
  const res: JournalsPage = await sdk.Journal.UserJournals(userID, meta, token)
  return { journals: res.journals ?? [], total: (res as any).total ?? 0 }
}

export async function listEntityJournals(
  entityType: string,
  entityId: string,
  domainId: string,
  token: string,
  page = 0,
): Promise<JournalsResult> {
  const meta: JournalsPageMetadata = { limit: JOURNAL_PAGE_SIZE, offset: page * JOURNAL_PAGE_SIZE }
  const res: JournalsPage = await sdk.Journal.EntityJournals(entityType, entityId, domainId, meta, token)
  return { journals: res.journals ?? [], total: (res as any).total ?? 0 }
}
