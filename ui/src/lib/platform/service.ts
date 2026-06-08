// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
import { getStoredAtomAccessToken } from '@/lib/auth/atom-auth-service'

export interface Workspace {
  id?: string
  name?: string
  route?: string | null
  status?: string | null
  createdAt?: string
  updatedAt?: string | null
}

export type Domain = Workspace

export interface WorkspaceMember {
  id: string
  name: string
  kind: string
  status: string
  tenantId?: string | null
  createdAt?: string
  updatedAt?: string | null
}

export interface WorkspaceInvitation {
  id: string
  tenantId: string
  inviteeUserId?: string | null
  inviteeEmail?: string | null
  invitedBy?: string
  roleId?: string | null
  roleName?: string | null
  acceptedAt?: string | null
  rejectedAt?: string | null
  revokedAt?: string | null
  createdAt?: string
  updatedAt?: string | null
}

export interface IdentityAuditLog {
  id: string
  entityId?: string | null
  tenantId?: string | null
  event: string
  outcome: string
  details: Record<string, unknown>
  createdAt: string
}

interface GraphQLError {
  message?: string
}

const atomAPIURL = import.meta.env.VITE_ATOM_API_URL ?? 'http://localhost:8080'
const atomGraphQLURL = import.meta.env.VITE_ATOM_GRAPHQL_URL ?? new URL('/graphql', atomAPIURL).toString()

async function graphQL<T>(query: string, variables?: Record<string, unknown>): Promise<T> {
  const token = getStoredAtomAccessToken()
  if (!token) throw new Error('ATOM session is missing.')

  const response = await fetch(atomGraphQLURL, {
    method: 'POST',
    credentials: 'omit',
    headers: {
      Accept: 'application/json',
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({ query, variables }),
  })

  const text = await response.text()
  const payload = text ? JSON.parse(text) as { data?: T; errors?: GraphQLError[] } : {}
  if (!response.ok) {
    throw new Error(payload.errors?.[0]?.message ?? `ATOM GraphQL request failed (${response.status}).`)
  }
  if (payload.errors?.length) {
    throw new Error(payload.errors[0]?.message ?? 'ATOM GraphQL request failed.')
  }
  if (!payload.data) {
    throw new Error('ATOM GraphQL returned an empty response.')
  }
  return payload.data
}

const TENANTS_QUERY = `
  query CubeWorkspaces($limit: Int = 100, $offset: Int = 0) {
    tenants(limit: $limit, offset: $offset) {
      items { id name route status createdAt updatedAt }
    }
  }
`

const CREATE_TENANT_MUTATION = `
  mutation CreateWorkspace($input: CreateTenantInput!) {
    createTenant(input: $input) {
      id
      name
      route
      status
      createdAt
      updatedAt
    }
  }
`

const DISABLE_TENANT_MUTATION = `
  mutation DisableWorkspace($id: ID!) {
    disableTenant(id: $id) { id status updatedAt }
  }
`

const MEMBERS_QUERY = `
  query CubeWorkspaceMembers($tenantId: ID!, $limit: Int = 100, $offset: Int = 0) {
    tenantMembers(tenantId: $tenantId, limit: $limit, offset: $offset) {
      items { id name kind status tenantId createdAt updatedAt }
      total
    }
  }
`

const REMOVE_MEMBER_MUTATION = `
  mutation RemoveWorkspaceMember($tenantId: ID!, $entityId: ID!) {
    removeTenantMember(tenantId: $tenantId, entityId: $entityId)
  }
`

const INVITATIONS_QUERY = `
  query CubeWorkspaceInvitations($tenantId: ID!, $limit: Int = 100, $offset: Int = 0) {
    tenantInvitations(tenantId: $tenantId, limit: $limit, offset: $offset) {
      items {
        id
        tenantId
        inviteeUserId
        inviteeEmail
        invitedBy
        roleId
        roleName
        acceptedAt
        rejectedAt
        revokedAt
        createdAt
        updatedAt
      }
      total
    }
  }
`

const MY_INVITATIONS_QUERY = `
  query CubeMyWorkspaceInvitations($limit: Int = 100, $offset: Int = 0) {
    myTenantInvitations(limit: $limit, offset: $offset) {
      items {
        id
        tenantId
        inviteeUserId
        inviteeEmail
        invitedBy
        roleId
        roleName
        acceptedAt
        rejectedAt
        revokedAt
        createdAt
        updatedAt
      }
      total
    }
  }
`

const CREATE_INVITATION_MUTATION = `
  mutation CreateWorkspaceInvitation($tenantId: ID!, $input: CreateTenantInvitationInput!) {
    createTenantInvitation(tenantId: $tenantId, input: $input) {
      id
      tenantId
      inviteeUserId
      inviteeEmail
      invitedBy
      roleId
      roleName
      acceptedAt
      rejectedAt
      revokedAt
      createdAt
      updatedAt
    }
  }
`

const REVOKE_INVITATION_MUTATION = `
  mutation RevokeWorkspaceInvitation($tenantId: ID!, $invitationId: ID!) {
    revokeTenantInvitation(tenantId: $tenantId, invitationId: $invitationId)
  }
`

const ACCEPT_INVITATION_MUTATION = `
  mutation AcceptWorkspaceInvitation($tenantId: ID!) {
    acceptTenantInvitation(tenantId: $tenantId)
  }
`

const REJECT_INVITATION_MUTATION = `
  mutation RejectWorkspaceInvitation($tenantId: ID!) {
    rejectTenantInvitation(tenantId: $tenantId)
  }
`

const AUDIT_LOGS_QUERY = `
  query CubeIdentityAuditLogs($tenantId: ID, $limit: Int = 50, $offset: Int = 0) {
    auditLogs(tenantId: $tenantId, limit: $limit, offset: $offset) {
      items { id entityId tenantId event outcome details createdAt }
      total
    }
  }
`

export async function listWorkspaces(): Promise<Workspace[]> {
  const data = await graphQL<{ tenants: { items: Workspace[] } }>(TENANTS_QUERY, { limit: 100, offset: 0 })
  return data.tenants.items ?? []
}

export async function createWorkspace(name: string, route: string): Promise<Workspace> {
  const data = await graphQL<{ createTenant: Workspace }>(CREATE_TENANT_MUTATION, {
    input: {
      name,
      route,
    },
  })
  return data.createTenant
}

export async function disableWorkspace(workspaceID: string): Promise<void> {
  await graphQL<{ disableTenant: Workspace }>(DISABLE_TENANT_MUTATION, { id: workspaceID })
}

export async function listWorkspaceMembers(workspaceID: string): Promise<WorkspaceMember[]> {
  const data = await graphQL<{ tenantMembers: { items: WorkspaceMember[] } }>(
    MEMBERS_QUERY,
    { tenantId: workspaceID, limit: 100, offset: 0 },
  )
  return data.tenantMembers.items ?? []
}

export async function removeWorkspaceMember(workspaceID: string, entityID: string): Promise<void> {
  await graphQL<{ removeTenantMember: boolean }>(REMOVE_MEMBER_MUTATION, {
    tenantId: workspaceID,
    entityId: entityID,
  })
}

export async function listWorkspaceInvitations(workspaceID: string): Promise<WorkspaceInvitation[]> {
  const data = await graphQL<{ tenantInvitations: { items: WorkspaceInvitation[] } }>(
    INVITATIONS_QUERY,
    { tenantId: workspaceID, limit: 100, offset: 0 },
  )
  return data.tenantInvitations.items ?? []
}

export async function listMyWorkspaceInvitations(): Promise<WorkspaceInvitation[]> {
  const data = await graphQL<{ myTenantInvitations: { items: WorkspaceInvitation[] } }>(
    MY_INVITATIONS_QUERY,
    { limit: 100, offset: 0 },
  )
  return data.myTenantInvitations.items ?? []
}

export async function createWorkspaceInvitation(workspaceID: string, inviteeEmail: string): Promise<WorkspaceInvitation> {
  const data = await graphQL<{ createTenantInvitation: WorkspaceInvitation }>(CREATE_INVITATION_MUTATION, {
    tenantId: workspaceID,
    input: {
      inviteeEmail,
    },
  })
  return data.createTenantInvitation
}

export async function revokeWorkspaceInvitation(workspaceID: string, invitationID: string): Promise<void> {
  await graphQL<{ revokeTenantInvitation: boolean }>(REVOKE_INVITATION_MUTATION, {
    tenantId: workspaceID,
    invitationId: invitationID,
  })
}

export async function acceptWorkspaceInvitation(workspaceID: string): Promise<void> {
  await graphQL<{ acceptTenantInvitation: boolean }>(ACCEPT_INVITATION_MUTATION, { tenantId: workspaceID })
}

export async function rejectWorkspaceInvitation(workspaceID: string): Promise<void> {
  await graphQL<{ rejectTenantInvitation: boolean }>(REJECT_INVITATION_MUTATION, { tenantId: workspaceID })
}

export async function listIdentityAuditLogs(workspaceID?: string): Promise<IdentityAuditLog[]> {
  const data = await graphQL<{ auditLogs: { items: IdentityAuditLog[] } }>(
    AUDIT_LOGS_QUERY,
    { tenantId: workspaceID || null, limit: 50, offset: 0 },
  )
  return data.auditLogs.items ?? []
}

export async function listDomains(_token?: string): Promise<Domain[]> {
  return listWorkspaces()
}

export async function listUserDomains(_userID: string, _token?: string): Promise<Domain[]> {
  return listWorkspaces()
}

export function toRoute(name: string): string {
  return name.trim().toLowerCase().replace(/\s+/g, '-').replace(/[^a-z0-9-]/g, '')
}

export async function createDomain(name: string, route: string, _token?: string): Promise<Domain> {
  return createWorkspace(name, route)
}

export async function getDomain(domainID: string, _token?: string): Promise<Domain> {
  const data = await graphQL<{ tenant: Workspace }>(
    `query CubeWorkspace($id: ID!) { tenant(id: $id) { id name route status createdAt updatedAt } }`,
    { id: domainID },
  )
  return data.tenant
}

export async function deleteDomain(domainID: string, _token?: string): Promise<void> {
  await disableWorkspace(domainID)
}
