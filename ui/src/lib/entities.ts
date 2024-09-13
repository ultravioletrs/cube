import { EntityType, type Metadata } from "@/types/entities";
import type { Domain, User } from "@absmach/magistrala-sdk";
import { DisableDomain, EnableDomain, UpdateDomain } from "./domains";
import { DisableUser, EnableUser, UpdateUser, UpdateUserTags } from "./users";

export async function UpdateMetadata(
  id: string,
  entity: EntityType,
  metadata: Metadata,
) {
  switch (entity) {
    case EntityType.User: {
      const user: User = {
        id: id,
        metadata: metadata,
      };
      return await UpdateUser(user);
    }
    case EntityType.Domain: {
      const domain: Domain = {
        id: id,
        metadata: metadata,
      };
      return await UpdateDomain(domain);
    }
    default: {
      return {
        data: null,
        error: "Invalid entity type",
      };
    }
  }
}

export async function UpdateTags(
  id: string,
  tags: string[],
  entity: EntityType,
) {
  switch (entity) {
    case EntityType.User: {
      const user: User = {
        id: id,
        tags: tags,
      };
      return await UpdateUserTags(user);
    }
    case EntityType.Domain: {
      const domain: Domain = {
        id: id,
        tags: tags,
      };
      return await UpdateDomain(domain);
    }
    default: {
      throw new Error("Invalid entity type");
    }
  }
}

export async function UpdateName(id: string, name: string, entity: EntityType) {
  switch (entity) {
    case EntityType.User: {
      const user: User = {
        id: id,
        name: name,
      };
      return await UpdateUser(user);
    }
    case EntityType.Domain: {
      const domain: Domain = {
        id: id,
        name: name,
      };
      return await UpdateDomain(domain);
    }
    default: {
      throw new Error("Invalid entity type");
    }
  }
}

export async function EnableEntity(id: string, entity: EntityType) {
  switch (entity) {
    case EntityType.User: {
      return await EnableUser(id);
    }
    case EntityType.Domain: {
      return await EnableDomain(id);
    }
    default: {
      return {
        data: null,
        error: "Invalid entity type",
      };
    }
  }
}

export async function DisableEntity(id: string, entity: EntityType) {
  switch (entity) {
    case EntityType.User: {
      return await DisableUser(id);
    }
    case EntityType.Domain: {
      return await DisableDomain(id);
    }
    default: {
      return {
        data: null,
        error: "Invalid entity type",
      };
    }
  }
}
