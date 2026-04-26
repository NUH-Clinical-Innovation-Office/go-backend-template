# API Reference

Base URL: `http://localhost:8080`

## Authentication

All authenticated endpoints require a Bearer token in the `Authorization` header:

```
Authorization: Bearer <token>
```

## Error Format

All error responses use a consistent JSON format:

```json
{
  "detail": "error message describing what went wrong"
}
```

## Status Codes

| Code | Description |
|------|-------------|
| 200  | OK - Request succeeded |
| 201  | Created - Resource created successfully |
| 204  | No Content - Resource deleted successfully |
| 400  | Bad Request - Invalid request body or parameters |
| 401  | Unauthorized - Missing or invalid authentication |
| 403  | Forbidden - Insufficient permissions (admin role required) |
| 404  | Not Found - Resource not found |
| 409  | Conflict - Resource already exists |
| 500  | Internal Server Error - Unexpected server error |
| 503  | Service Unavailable - Health check failed (database disconnected) |

---

## Public Endpoints

### GET /

Returns API version and status.

**Response** `200 OK`

```json
{
  "version": "1.0.0",
  "status": "running"
}
```

---

### GET /health

Returns API health status including database connectivity check.

**Response** `200 OK` (healthy)

```json
{
  "status": "healthy"
}
```

**Response** `503 Service Unavailable` (unhealthy)

```json
{
  "status": "unhealthy"
}
```

---

## Authentication (Public)

### POST /api/v1/auth/register

Register a new user account. Requires a valid `approved_id` that corresponds to an approved user in the system.

**Request Body**

```json
{
  "email": "user@example.com",
  "password": "SecurePass123",
  "approved_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

| Field       | Type   | Required | Description |
|-------------|--------|----------|-------------|
| email       | string | Yes      | Valid email address |
| password    | string | Yes      | Min 8 chars, requires uppercase, lowercase, and digit |
| approved_id | string | Yes      | UUID of an approved user |

**Response** `201 Created`

```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "bearer"
}
```

**Error Responses**

- `400 Bad Request` - Invalid request body or validation failure
  ```json
  {"detail": "email is required"}
  {"detail": "password must contain at least one uppercase letter"}
  ```
- `404 Not Found` - Approved user not found
  ```json
  {"detail": "approved user not found"}
  ```
- `409 Conflict` - User already exists
  ```json
  {"detail": "user already exists"}
  ```
- `500 Internal Server Error`
  ```json
  {"detail": "internal server error"}
  ```

---

### POST /api/v1/auth/login

Login with email and password credentials.

**Request Body**

```json
{
  "email": "user@example.com",
  "password": "SecurePass123"
}
```

| Field    | Type   | Required | Description |
|----------|--------|----------|-------------|
| email    | string | Yes      | Valid email address |
| password | string | Yes      | Account password |

**Response** `200 OK`

```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "bearer"
}
```

**Error Responses**

- `400 Bad Request` - Invalid request body or validation failure
  ```json
  {"detail": "email is required"}
  ```
- `401 Unauthorized` - Invalid credentials
  ```json
  {"detail": "invalid credentials"}
  ```
- `500 Internal Server Error`
  ```json
  {"detail": "internal server error"}
  ```

---

## Authenticated Endpoints

### GET /api/v1/me

Get the current authenticated user's profile.

**Headers**

```
Authorization: Bearer <token>
```

**Response** `200 OK`

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "user@example.com",
  "first_name": "John",
  "is_active": true,
  "roles": ["user"],
  "created_at": "2024-01-01T00:00:00Z"
}
```

| Field      | Type    | Description |
|------------|---------|-------------|
| id         | string  | User UUID |
| email      | string  | User email (from approved user) |
| first_name | string  | User first name (from approved user) |
| is_active  | boolean | Whether the user account is active |
| roles      | array   | User roles (e.g., "user", "admin") |
| created_at | string  | ISO 8601 timestamp |

**Error Responses**

- `401 Unauthorized` - Missing or invalid token
  ```json
  {"detail": "missing authorization header"}
  {"detail": "could not validate credentials"}
  ```
- `500 Internal Server Error`
  ```json
  {"detail": "internal server error"}
  ```

---

## Todos (User-Scoped)

All todo endpoints are scoped to the authenticated user. Users can only access their own todos.

### GET /api/v1/todos/

List all todos for the authenticated user.

**Headers**

```
Authorization: Bearer <token>
```

**Response** `200 OK`

```json
[
  {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "user_id": "660e8400-e29b-41d4-a716-446655440001",
    "title": "Complete project",
    "description": "Finish the backend API implementation",
    "is_completed": false,
    "due_date": "2024-12-31T23:59:59Z",
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
]
```

**Todo Object**

| Field       | Type     | Description |
|------------|----------|-------------|
| id         | string   | Todo UUID |
| user_id    | string   | Owner's user UUID |
| title      | string   | Todo title (max 500 chars) |
| description| string?  | Optional description |
| is_completed| boolean | Completion status |
| due_date   | string?  | ISO 8601 timestamp (RFC3339), optional |
| created_at | string   | ISO 8601 timestamp |
| updated_at | string   | ISO 8601 timestamp |

**Error Responses**

- `401 Unauthorized` - Missing or invalid token
- `500 Internal Server Error`

---

### POST /api/v1/todos/

Create a new todo for the authenticated user.

**Headers**

```
Authorization: Bearer <token>
Content-Type: application/json
```

**Request Body**

```json
{
  "title": "Complete project",
  "description": "Finish the backend API implementation",
  "due_date": "2024-12-31T23:59:59Z"
}
```

| Field       | Type   | Required | Description |
|-------------|--------|----------|-------------|
| title       | string | Yes     | Todo title (max 500 chars) |
| description | string | No      | Optional description |
| due_date    | string | No      | ISO 8601 timestamp (RFC3339) |

**Response** `201 Created`

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "user_id": "660e8400-e29b-41d4-a716-446655440001",
  "title": "Complete project",
  "description": "Finish the backend API implementation",
  "is_completed": false,
  "due_date": "2024-12-31T23:59:59Z",
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

**Error Responses**

- `400 Bad Request` - Invalid request body or validation failure
  ```json
  {"detail": "title is required"}
  {"detail": "title must be less than 500 characters"}
  ```
- `401 Unauthorized` - Missing or invalid token
- `500 Internal Server Error`

---

### GET /api/v1/todos/{id}

Get a single todo by ID. The todo must belong to the authenticated user.

**Headers**

```
Authorization: Bearer <token>
```

**Path Parameters**

| Parameter | Type   | Description |
|-----------|--------|-------------|
| id        | string | Todo UUID |

**Response** `200 OK`

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "user_id": "660e8400-e29b-41d4-a716-446655440001",
  "title": "Complete project",
  "description": "Finish the backend API implementation",
  "is_completed": false,
  "due_date": "2024-12-31T23:59:59Z",
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

**Error Responses**

- `400 Bad Request` - Invalid ID format
  ```json
  {"detail": "id is required"}
  {"detail": "invalid id format"}
  ```
- `401 Unauthorized` - Missing or invalid token
- `404 Not Found` - Todo not found or does not belong to user
  ```json
  {"detail": "todo not found"}
  ```
- `500 Internal Server Error`

---

### PUT /api/v1/todos/{id}

Update a todo. The todo must belong to the authenticated user.

**Headers**

```
Authorization: Bearer <token>
Content-Type: application/json
```

**Path Parameters**

| Parameter | Type   | Description |
|-----------|--------|-------------|
| id        | string | Todo UUID |

**Request Body**

```json
{
  "title": "Updated title",
  "description": "Updated description",
  "is_completed": true,
  "due_date": "2024-12-31T23:59:59Z"
}
```

| Field       | Type    | Required | Description |
|-------------|---------|----------|-------------|
| title       | string  | Yes      | Todo title (max 500 chars) |
| description | string  | No      | Optional description (set to null to clear) |
| is_completed| boolean | No      | Completion status (defaults to false) |
| due_date    | string  | No      | ISO 8601 timestamp (RFC3339) |

**Response** `200 OK`

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "user_id": "660e8400-e29b-41d4-a716-446655440001",
  "title": "Updated title",
  "description": "Updated description",
  "is_completed": true,
  "due_date": "2024-12-31T23:59:59Z",
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-02T00:00:00Z"
}
```

**Error Responses**

- `400 Bad Request` - Invalid request body, ID format, or validation failure
- `401 Unauthorized` - Missing or invalid token
- `404 Not Found` - Todo not found or does not belong to user
  ```json
  {"detail": "todo not found"}
  ```
- `500 Internal Server Error`

---

### DELETE /api/v1/todos/{id}

Delete a todo. The todo must belong to the authenticated user.

**Headers**

```
Authorization: Bearer <token>
```

**Path Parameters**

| Parameter | Type   | Description |
|-----------|--------|-------------|
| id        | string | Todo UUID |

**Response** `204 No Content`

**Error Responses**

- `400 Bad Request` - Invalid ID format
  ```json
  {"detail": "id is required"}
  {"detail": "invalid id format"}
  ```
- `401 Unauthorized` - Missing or invalid token
- `404 Not Found` - Todo not found or does not belong to user
  ```json
  {"detail": "todo not found"}
  ```
- `500 Internal Server Error`

---

## Admin Endpoints

All admin endpoints require the `admin` role in the authenticated user's token.

### GET /api/v1/admin/approved-users/

List all approved users in the system.

**Headers**

```
Authorization: Bearer <admin-token>
```

**Response** `200 OK`

```json
[
  {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "user@example.com",
    "first_name": "John",
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
]
```

**Approved User Object**

| Field      | Type   | Description |
|------------|--------|-------------|
| id         | string | Approved user UUID |
| email      | string | Approved email address |
| first_name | string | First name |
| created_at | string | ISO 8601 timestamp |
| updated_at | string | ISO 8601 timestamp |

**Error Responses**

- `401 Unauthorized` - Missing or invalid token
- `403 Forbidden` - Admin role required
  ```json
  {"detail": "admin role required"}
  ```
- `500 Internal Server Error`

---

### POST /api/v1/admin/approved-users/

Create a new approved user. The creating user must have admin role.

**Headers**

```
Authorization: Bearer <admin-token>
Content-Type: application/json
```

**Request Body**

```json
{
  "email": "newuser@example.com",
  "first_name": "Jane"
}
```

| Field      | Type   | Required | Description |
|------------|--------|----------|-------------|
| email      | string | Yes      | Valid email address |
| first_name | string | Yes      | First name (letters, spaces, hyphens, apostrophes only) |

**Response** `201 Created`

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "newuser@example.com",
  "first_name": "Jane",
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

**Error Responses**

- `400 Bad Request` - Invalid request body or validation failure
  ```json
  {"detail": "email is required"}
  {"detail": "invalid email format"}
  {"detail": "first name is required"}
  ```
- `401 Unauthorized` - Missing or invalid token
- `403 Forbidden` - Admin role required
- `409 Conflict` - Email already in approved list
  ```json
  {"detail": "Email already in approved list"}
  ```
- `500 Internal Server Error`

---

### POST /api/v1/admin/approved-users/bulk

Create multiple approved users in a single request.

**Headers**

```
Authorization: Bearer <admin-token>
Content-Type: application/json
```

**Request Body**

```json
{
  "users": [
    {"email": "user1@example.com", "first_name": "User"},
    {"email": "user2@example.com", "first_name": "Another"}
  ]
}
```

| Field | Type  | Required | Description |
|-------|-------|----------|-------------|
| users | array | Yes      | Array of approved user objects (min 1) |

**User Object** (within users array)

| Field      | Type   | Required | Description |
|------------|--------|----------|-------------|
| email      | string | Yes      | Valid email address |
| first_name | string | Yes      | First name |

**Response** `201 Created`

```json
[
  {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "user1@example.com",
    "first_name": "User",
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  },
  {
    "id": "660e8400-e29b-41d4-a716-446655440001",
    "email": "user2@example.com",
    "first_name": "Another",
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
]
```

**Error Responses**

- `400 Bad Request` - Invalid request body or users array empty
  ```json
  {"detail": "users array is required"}
  ```
- `401 Unauthorized` - Missing or invalid token
- `403 Forbidden` - Admin role required
- `500 Internal Server Error`

---

### DELETE /api/v1/admin/approved-users/{id}

Delete an approved user by ID.

**Headers**

```
Authorization: Bearer <admin-token>
```

**Path Parameters**

| Parameter | Type   | Description |
|-----------|--------|-------------|
| id        | string | Approved user UUID |

**Response** `204 No Content`

**Error Responses**

- `400 Bad Request` - Invalid ID format
  ```json
  {"detail": "id is required"}
  {"detail": "invalid id format"}
  ```
- `401 Unauthorized` - Missing or invalid token
- `403 Forbidden` - Admin role required
- `500 Internal Server Error`

---

## Validation Rules

### Email

- Required
- Must be a valid email format

### Password

- Required
- Minimum 8 characters
- Must contain at least one uppercase letter
- Must contain at least one lowercase letter
- Must contain at least one digit

### Todo Title

- Required
- Maximum 500 characters
- Leading/trailing whitespace is trimmed

### First Name

- Required
- Only letters, spaces, hyphens (`-`), and apostrophes (`'`) allowed

### Due Date

- Must be a valid ISO 8601 timestamp in RFC3339 format (e.g., `2024-12-31T23:59:59Z`)
- Invalid formats are silently ignored and stored as null

---

## Headers Reference

### Standard Request Headers

| Header           | Value                          | Required |
|------------------|--------------------------------|----------|
| Authorization    | Bearer \<token\>               | For protected routes |
| Content-Type    | application/json               | For POST/PUT requests |
| X-Request-ID    | UUID                           | Optional (auto-generated if not provided) |

### Standard Response Headers

| Header                  | Value                                    |
|-------------------------|------------------------------------------|
| Content-Type            | application/json                         |
| X-Request-ID           | Unique request identifier                 |
| Access-Control-Allow-Origin | * (or configured origin)           |
