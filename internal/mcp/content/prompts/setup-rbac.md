Guide for configuring authentication and role-based access control (RBAC).

## Authentication Methods

The registry supports multiple auth methods (configured in security.auth.methods):

| Method | Description |
|--------|-------------|
| **basic** | Username/password via HTTP Basic Auth |
| **api_key** | API keys sent as Bearer tokens |
| **jwt** | JWT tokens (for external identity providers) |
| **oidc** | OpenID Connect (delegated to an OIDC provider) |
| **ldap** | LDAP directory authentication |
| **mtls** | Mutual TLS client certificates |

## The 4 Built-in Roles

| Role | Permissions |
|------|-------------|
| **admin** | Full access: manage users, API keys, schemas, config, modes |
| **write** | Register/delete schemas, set config/mode, manage encryption keys |
| **read** | Read schemas, subjects, config, mode. Cannot modify anything. |
| **readwrite** | Read + write schemas and config. Cannot manage users or API keys. |

## Setup Steps

### Step 1: Enable auth in config
Set security.auth.enabled: true and choose methods.

### Step 2: Create admin user
Use **create_user** with role: admin.

### Step 3: Create service accounts
Use **create_api_key** for each service:
- Producers: role write or readwrite
- Consumers: role read
- CI/CD: role write (for schema registration)
- Monitoring: role read

### Step 4: Test access
Use **list_users** and **list_api_keys** to verify.

## MCP Admin Tools

- **create_user / get_user / list_users / update_user / delete_user** -- manage users
- **create_api_key / get_api_key / list_api_keys / update_api_key / delete_api_key** -- manage API keys
- **list_roles** -- list available roles and their permissions
