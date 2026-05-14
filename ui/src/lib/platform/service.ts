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

// The magistrala SDK throws plain objects { status, error } rather than Error
// instances. This converts them so callers can use err.message normally.
function sdkError(err: unknown): Error {
  if (err instanceof Error) return err
  if (err && typeof err === 'object' && 'error' in err) {
    const e = err as { error: unknown; status?: unknown }
    const msg = typeof e.error === 'string' ? e.error : JSON.stringify(e.error)
    return new Error(msg || `request failed (${e.status ?? 'unknown'})`)
  }
  return new Error(String(err))
}

const sdk = new SDK({
  usersUrl: window.location.origin,
  domainsUrl: window.location.origin,
  journalUrl: window.location.origin,
})

// ── Domains ────────────────────────────────────────────────────────────────

export async function listDomains(token: string): Promise<Domain[]> {
  try {
    const page: DomainsPage = await sdk.Domains.Domains({ limit: 100, offset: 0 } as PageMetadata, token)
    return page.domains ?? []
  } catch (e) { throw sdkError(e) }
}

export async function listUserDomains(userID: string, token: string): Promise<Domain[]> {
  try {
    const page: DomainsPage = await sdk.Domains.ListUserDomains(userID, { limit: 100, offset: 0 } as PageMetadata, token)
    return page.domains ?? []
  } catch (e) { throw sdkError(e) }
}

export function toRoute(name: string): string {
  return name.trim().toLowerCase().replace(/\s+/g, '-').replace(/[^a-z0-9-]/g, '')
}

export async function createDomain(name: string, route: string, token: string): Promise<Domain> {
  try {
    return await sdk.Domains.CreateDomain({ name, route } as Domain, token)
  } catch (e) { throw sdkError(e) }
}

export async function getDomain(domainID: string, token: string): Promise<Domain> {
  try {
    return await sdk.Domains.Domain(domainID, token)
  } catch (e) { throw sdkError(e) }
}

export async function deleteDomain(domainID: string, token: string): Promise<void> {
  try {
    await sdk.Domains.DisableDomain(domainID, token)
  } catch (e) { throw sdkError(e) }
}

// ── Members ────────────────────────────────────────────────────────────────

export interface MembersResult {
  members: MemberRoles[]
  total: number
}

export async function listDomainMembers(domainID: string, token: string): Promise<MembersResult> {
  try {
    const page: MemberRolesPage = await sdk.Domains.ListDomainMembers(
      domainID,
      { limit: 100, offset: 0 } as BasicPageMeta,
      token,
    )
    return { members: page.members ?? [], total: page.total ?? 0 }
  } catch (e) { throw sdkError(e) }
}

// ── Users ──────────────────────────────────────────────────────────────────

export async function listUsers(token: string): Promise<User[]> {
  try {
    const page: UsersPage = await sdk.Users.Users({ limit: 100, offset: 0 } as PageMetadata, token)
    return page.users ?? []
  } catch (e) { throw sdkError(e) }
}

export async function searchUsers(query: string, token: string): Promise<User[]> {
  try {
    const page: UsersPage = await sdk.Users.SearchUsers({ limit: 20, offset: 0, name: query } as PageMetadata, token)
    return page.users ?? []
  } catch (e) { throw sdkError(e) }
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
  try {
    const page: InvitationsPage = await sdk.Domains.ListDomainInvitations(
      { limit: 100, offset: 0 } as InvitationPageMeta,
      domainID,
      token,
    )
    return { invitations: page.invitations ?? [], total: page.total ?? 0 }
  } catch (e) { throw sdkError(e) }
}

export async function listUserInvitations(token: string): Promise<InvitationsResult> {
  try {
    const page: InvitationsPage = await sdk.Domains.ListUserInvitations(
      { limit: 100, offset: 0 } as PageMetadata,
      token,
    )
    return { invitations: page.invitations ?? [], total: page.total ?? 0 }
  } catch (e) { throw sdkError(e) }
}

export async function sendInvitation(userID: string, domainID: string, roleID: string, token: string): Promise<void> {
  try {
    await sdk.Domains.SendInvitation(userID, domainID, roleID, token)
  } catch (e) { throw sdkError(e) }
}

export async function acceptInvitation(domainID: string, token: string): Promise<void> {
  try {
    await sdk.Domains.AcceptInvitation(domainID, token)
  } catch (e) { throw sdkError(e) }
}

export async function rejectInvitation(domainID: string, token: string): Promise<void> {
  try {
    await sdk.Domains.RejectInvitation(domainID, token)
  } catch (e) { throw sdkError(e) }
}

export async function deleteInvitation(userID: string, domainID: string, token: string): Promise<void> {
  try {
    await sdk.Domains.DeleteInvitation(userID, domainID, token)
  } catch (e) { throw sdkError(e) }
}

// ── Journal / Audit Logs ───────────────────────────────────────────────────

export interface JournalsResult {
  journals: Journal[]
  total: number
}

export const JOURNAL_PAGE_SIZE = 10

export async function listUserJournals(userID: string, token: string, page = 0): Promise<JournalsResult> {
  try {
    const meta: JournalsPageMetadata = { limit: JOURNAL_PAGE_SIZE, offset: page * JOURNAL_PAGE_SIZE }
    const res: JournalsPage = await sdk.Journal.UserJournals(userID, meta, token)
    return { journals: res.journals ?? [], total: (res as any).total ?? 0 }
  } catch (e) { throw sdkError(e) }
}

export async function listEntityJournals(
  entityType: string,
  entityId: string,
  domainId: string,
  token: string,
  page = 0,
): Promise<JournalsResult> {
  try {
    const meta: JournalsPageMetadata = { limit: JOURNAL_PAGE_SIZE, offset: page * JOURNAL_PAGE_SIZE }
    const res: JournalsPage = await sdk.Journal.EntityJournals(entityType, entityId, domainId, meta, token)
    return { journals: res.journals ?? [], total: (res as any).total ?? 0 }
  } catch (e) { throw sdkError(e) }
}
