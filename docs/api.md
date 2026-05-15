# API Reference

## Base URL

```
http://localhost:8080
```

## Authentication

Most endpoints require JWT Bearer token authentication:

```
Authorization: Bearer <token>
```

Admin endpoints require the `admin` role in the JWT claims.

## Public Endpoints

### Health Check

```
GET /health
```

Returns API health status with database connectivity check.

**Response (200 - healthy):**
```json
{
  "status": "healthy"
}
```

**Response (503 - unhealthy):**
```json
{
  "status": "unhealthy"
}
```

### Root

```
GET /
```

Returns API version and status.

**Response:**
```json
{
  "version": "1.0.0",
  "status": "running"
}
```

---

## Authentication

### Register

```
POST /api/v1/auth/register
```

Register a new user. Only users with pre-approved emails can register.

**Request Body:**
```json
{
  "email": "user@example.com",
  "password": "securePassword123"
}
```

**Response (201):**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "token_type": "bearer"
}
```

**Errors:**
- `400` - Invalid request body
- `401` - Email not approved
- `409` - Email already registered

---

### Login

```
POST /api/v1/auth/login
```

Authenticate and receive a JWT token.

**Request Body:**
```json
{
  "email": "user@example.com",
  "password": "securePassword123"
}
```

**Response (200):**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "token_type": "bearer"
}
```

**Errors:**
- `400` - Invalid request body
- `401` - Invalid credentials

---

## User (Authenticated)

### Get Current User

```
GET /api/v1/me
```

Get the currently authenticated user.

**Headers:**
```
Authorization: Bearer <token>
```

**Response (200):**
```json
{
  "id": "uuid",
  "email": "user@example.com",
  "first_name": "John",
  "is_active": true,
  "roles": ["user"],
  "created_at": "2024-01-01T00:00:00Z"
}
```

**Errors:**
- `401` - Unauthorized (missing or invalid token)

---

## Todos

All todo endpoints require authentication.

### List Todos

```
GET /api/v1/todos
```

**Headers:**
```
Authorization: Bearer <token>
```

**Response (200):** Returns an array of todo objects.

```json
[
  {
    "id": "uuid",
    "user_id": "uuid",
    "title": "Todo title",
    "description": "Optional description",
    "is_completed": false,
    "due_date": "2024-01-01T00:00:00Z",
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
]
```

---

### Create Todo

```
POST /api/v1/todos
```

**Headers:**
```
Authorization: Bearer <token>
```

**Request Body:**
```json
{
  "title": "New todo",
  "description": "Optional description",
  "due_date": "2024-01-01T00:00:00Z"
}
```

**Response (201):** Returns the created todo object directly (not wrapped).

```json
{
  "id": "uuid",
  "user_id": "uuid",
  "title": "New todo",
  "description": "Optional description",
  "is_completed": false,
  "due_date": "2024-01-01T00:00:00Z",
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

---

### Get Todo

```
GET /api/v1/todos/{id}
```

**Headers:**
```
Authorization: Bearer <token>
```

**Response (200):** Returns the todo object directly (not wrapped).

```json
{
  "id": "uuid",
  "user_id": "uuid",
  "title": "Todo title",
  "description": "Optional description",
  "is_completed": false,
  "due_date": "2024-01-01T00:00:00Z",
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

**Errors:**
- `400` - Invalid id format
- `404` - Todo not found

---

### Update Todo

```
PUT /api/v1/todos/{id}
```

**Headers:**
```
Authorization: Bearer <token>
```

**Request Body:**
```json
{
  "title": "Updated title",
  "description": "Updated description",
  "is_completed": true,
  "due_date": "2024-01-01T00:00:00Z"
}
```

**Response (200):** Returns the updated todo object directly (not wrapped).

```json
{
  "id": "uuid",
  "user_id": "uuid",
  "title": "Updated title",
  "description": "Updated description",
  "is_completed": true,
  "due_date": "2024-01-01T00:00:00Z",
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

**Errors:**
- `400` - Invalid id format
- `404` - Todo not found

---

### Delete Todo

```
DELETE /api/v1/todos/{id}
```

**Headers:**
```
Authorization: Bearer <token>
```

**Response (204):** No content

**Errors:**
- `400` - Invalid id format
- `404` - Todo not found

---

## Admin (Admin Role Required)

All admin endpoints require `Authorization: Bearer <token>` with admin role.

### List Approved Users

```
GET /api/v1/admin/approved-users
```

**Headers:**
```
Authorization: Bearer <token>
```

**Response (200):**
```json
{
  "approved_users": [
    {
      "id": "uuid",
      "email": "approved@example.com",
      "created_at": "2024-01-01T00:00:00Z"
    }
  ]
}
```

---

### Create Approved User

```
POST /api/v1/admin/approved-users
```

**Headers:**
```
Authorization: Bearer <token>
```

**Request Body:**
```json
{
  "email": "newuser@example.com"
}
```

**Response (201):**
```json
{
  "approved_user": {
    "id": "uuid",
    "email": "newuser@example.com",
    "created_at": "2024-01-01T00:00:00Z"
  }
}
```

---

### Bulk Create Approved Users

```
POST /api/v1/admin/approved-users/bulk
```

**Headers:**
```
Authorization: Bearer <token>
```

**Request Body:**
```json
{
  "emails": [
    "user1@example.com",
    "user2@example.com"
  ]
}
```

**Response (201):**
```json
{
  "approved_users": [
    {
      "id": "uuid",
      "email": "user1@example.com",
      "created_at": "2024-01-01T00:00:00Z"
    },
    {
      "id": "uuid",
      "email": "user2@example.com",
      "created_at": "2024-01-01T00:00:00Z"
    }
  ]
}
```

---

### Delete Approved User

```
DELETE /api/v1/admin/approved-users/{id}
```

**Headers:**
```
Authorization: Bearer <token>
```

**Response (204):** No content

**Errors:**
- `404` - Approved user not found

---

## Error Response Format

All errors return a JSON body:

```json
{
  "error": "Error message here"
}
```

Common HTTP status codes:
- `400` - Bad Request
- `401` - Unauthorized
- `403` - Forbidden
- `404` - Not Found
- `409` - Conflict
- `500` - Internal Server Error