# Router Package

The `router` package provides a priority-based routing engine for the Cube proxy, enabling sophisticated request matching and forwarding to configurable target URLs.

## Overview

The router matches incoming HTTP requests against a set of configurable rules and determines the appropriate target URL. Routes are evaluated in priority order, with support for pattern matching on multiple request attributes (path, method, headers, query parameters, and request body).

## Core Concepts

### Route Rule

A `RouteRule` defines a complete routing configuration with the following properties:

- **Name**: Unique identifier for the route (must be alphanumeric with hyphens or underscores)
- **TargetURL**: The destination URL where matching requests are forwarded
- **Matchers**: Conditions that must all match for the route to be selected (AND logic)
- **Priority**: Determines evaluation order; higher values are checked first (0-1000)
- **DefaultRule**: If true, matches when no other rules match
- **StripPrefix**: Optional path prefix to remove before forwarding
- **Enabled**: Optional flag to skip disabled routes (defaults to true)

### Route Matching

Routes are evaluated in descending priority order. All matchers within a route must match (AND logic) for the route to be selected. The router supports matching on:

- **Path**: Exact or regex patterns
- **HTTP Method**: GET, POST, PUT, etc.
- **Headers**: Custom header values
- **Query Parameters**: URL query string values
- **Request Body**: Field values or regex patterns in JSON bodies

## Route Management API

### Creating Routes

Create a new routing rule with a unique name and configuration:

```
POST /api/v1/routes
```

Validation ensures:
- Route name is unique and properly formatted
- At least one matcher is configured (unless it's a default rule)
- Target URL is valid
- Priority is between 0 and 1000
- Only one default route exists per configuration

### Updating Routes - Rename Semantics

The update operation supports renaming routes through its dual-parameter design:

```
PUT /api/v1/routes/{name}
```

**Important: Rename Safety**

- The `{name}` path parameter specifies the **current** route name to identify which route to update
- The `name` field in the request body specifies the **new** route name
- If the request body name differs from the path name, the route is renamed
- If they match, the route is updated without renaming

**Example: Renaming a route from "api-v1" to "api-v2":**

```bash
curl -X PUT /api/v1/routes/api-v1 \
  -H "Content-Type: application/json" \
  -d '{
    "name": "api-v2",
    "target_url": "http://backend:8080",
    "matchers": [...],
    "priority": 10
  }'
```

**Example: Updating a route without renaming:**

```bash
curl -X PUT /api/v1/routes/api-v1 \
  -H "Content-Type: application/json" \
  -d '{
    "name": "api-v1",
    "target_url": "http://backend:9000",
    "matchers": [...],
    "priority": 20
  }'
```

### Querying Routes

Retrieve a specific routing rule:

```
GET /api/v1/routes/{name}
```

List all active routes with pagination:

```
GET /api/v1/routes?offset=0&limit=50
```

### Deleting Routes

Remove a routing rule:

```
DELETE /api/v1/routes/{name}
```

## Safety Considerations

### Atomic Updates

Route updates are atomic at the database level, but clients should be aware that:

- In-flight requests may use old route configurations during updates
- Route priority changes take effect immediately for new requests
- Renaming a route invalidates any cached references to the old name

### System Routes

Certain routes are protected and cannot be modified or deleted. These include:

- Cube internal routes and audit routes
- Any route marked as a system route in the configuration

Attempts to create, update, or delete system routes will return `ErrSystemRouteProtected`.

### Validation Rules

Route names must:
- Be non-empty
- Contain only alphanumeric characters, hyphens (-), and underscores (_)
- Be unique within the routing configuration

All matchers must define valid patterns. Regex patterns are validated at route creation/update time.

### Conflicts and Constraints

- **Duplicate Names**: Creating or renaming to an existing route name will fail
- **Multiple Defaults**: Only one default route is allowed
- **Disabled Routes**: Routes with `enabled: false` are excluded from matching but still stored
- **Invalid Patterns**: Malformed regex in matchers will be rejected with `ErrInvalidRegex`

### Request Forwarding

After a route is matched:

1. The matched route's target URL is selected
2. If `StripPrefix` is configured, it's removed from the request path
3. The request is forwarded to the target URL
4. Response is returned to the client

## Configuration Example

```json
{
  "routes": [
    {
      "name": "api-v2",
      "target_url": "http://backend-v2:8080",
      "matchers": [
        {
          "condition": "path",
          "pattern": "^/api/v2/.*",
          "is_regex": true
        },
        {
          "condition": "method",
          "pattern": "GET|POST"
        }
      ],
      "priority": 100,
      "enabled": true
    },
    {
      "name": "api-v1",
      "target_url": "http://backend-v1:8080",
      "matchers": [
        {
          "condition": "path",
          "pattern": "^/api/v1/.*",
          "is_regex": true
        }
      ],
      "priority": 50,
      "enabled": true
    },
    {
      "name": "default",
      "target_url": "http://default-backend:8080",
      "matchers": [],
      "default_rule": true,
      "priority": 0
    }
  ],
  "default_url": "http://fallback:8080"
}
```

## Client Implementation Tips

### When Renaming Routes

1. **Always verify the current name** before performing a rename operation
2. **Ensure unique new names** by checking existing routes first
3. **Handle concurrent updates** gracefully; check for `ErrRouteConflict`
4. **Update internal references** if your client caches route names

### Monitoring Route Changes

- Subscribe to route update events if available
- Reload route configurations periodically
- Log rename operations for audit trails

### Error Handling

Expect these common errors:

- `ErrRouteNotFound`: Route doesn't exist (may occur if renamed elsewhere)
- `ErrRouteConflict`: Name already exists
- `ErrInvalidRouteName`: Name format is invalid
- `ErrSystemRouteProtected`: Cannot modify protected routes
- `ErrMultipleDefaultRoutes`: Only one default route allowed

## Performance Considerations

- **Priority-based evaluation**: Higher priority routes are checked first, so frequently matched routes should have higher priority
- **Regex performance**: Complex regex patterns in matchers impact routing latency
- **Memory usage**: Each route maintains a compiled matcher in memory; disable unused routes with `enabled: false`
- **Atomic updates**: Route configuration updates are thread-safe but may cause brief lock contention on high-traffic systems
