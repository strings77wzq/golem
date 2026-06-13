# Spec: Database Schema

## [S1] Schema Definition

Three tables with proper relationships:

### users
| Column | Type | Constraints |
|--------|------|-------------|
| id | INTEGER | PRIMARY KEY AUTOINCREMENT |
| username | TEXT | NOT NULL UNIQUE |
| email | TEXT | NOT NULL UNIQUE |
| password_hash | TEXT | NOT NULL |
| display_name | TEXT | nullable |
| avatar_url | TEXT | nullable |
| bio | TEXT | nullable |
| created_at | DATETIME | DEFAULT CURRENT_TIMESTAMP |
| updated_at | DATETIME | DEFAULT CURRENT_TIMESTAMP |

### articles
| Column | Type | Constraints |
|--------|------|-------------|
| id | INTEGER | PRIMARY KEY AUTOINCREMENT |
| author_id | INTEGER | NOT NULL, FK → users(id) ON DELETE CASCADE |
| title | TEXT | NOT NULL |
| slug | TEXT | NOT NULL UNIQUE |
| content | TEXT | NOT NULL |
| summary | TEXT | nullable |
| status | TEXT | DEFAULT 'draft', CHECK IN ('draft', 'published', 'archived') |
| view_count | INTEGER | DEFAULT 0 |
| created_at | DATETIME | DEFAULT CURRENT_TIMESTAMP |
| updated_at | DATETIME | DEFAULT CURRENT_TIMESTAMP |
| published_at | DATETIME | nullable |

### comments
| Column | Type | Constraints |
|--------|------|-------------|
| id | INTEGER | PRIMARY KEY AUTOINCREMENT |
| article_id | INTEGER | NOT NULL, FK → articles(id) ON DELETE CASCADE |
| user_id | INTEGER | nullable, FK → users(id) ON DELETE SET NULL |
| parent_id | INTEGER | nullable, FK → comments(id) ON DELETE CASCADE |
| author_name | TEXT | nullable (for anonymous comments) |
| content | TEXT | NOT NULL |
| created_at | DATETIME | DEFAULT CURRENT_TIMESTAMP |

## [S2] SchemaManager

The `SchemaManager` handles schema creation and introspection.

**Methods:**
- `InitSchema(ctx)` — CREATE TABLE IF NOT EXISTS for all tables
- `GetSchema(ctx)` — return full schema as string
- `GetTableInfo(ctx, table)` — return column info for one table
- `GetTables(ctx)` — return list of table names

**Constraints:**
- Must be idempotent (safe to call multiple times)
- Must use `modernc.org/sqlite` (pure Go, no CGO)
- Schema stored in configurable path (default: `~/.golem/data.db`)

## [S3] Seed Data

Initial seed data for demonstration:

```sql
INSERT INTO users (username, email, password_hash, display_name, bio)
VALUES
  ('admin', 'admin@example.com', 'hash1', 'Admin', 'System administrator'),
  ('alice', 'alice@example.com', 'hash2', 'Alice', 'Tech writer'),
  ('bob', 'bob@example.com', 'hash3', 'Bob', 'Developer');

INSERT INTO articles (author_id, title, slug, content, summary, status, published_at)
VALUES
  (1, 'Golem v0.6.0 Release', 'golem-v060-release', 'Content...', 'New release', 'published', NOW),
  (2, 'Getting Started with Go Agents', 'go-agents-guide', 'Content...', 'Guide', 'published', NOW),
  (3, 'Database Design Patterns', 'db-design-patterns', 'Content...', 'Patterns', 'draft', NULL);

INSERT INTO comments (article_id, user_id, author_name, content)
VALUES
  (1, 2, NULL, 'Great release!'),
  (1, 3, NULL, 'Looking forward to the next version'),
  (2, 1, NULL, 'Very helpful guide');
```

## [S4] Schema as String

The `GetSchema()` method returns a human-readable string for injection into system prompt:

```
Database: SQLite

Tables:
- users (id, username, email, display_name, bio, created_at)
- articles (id, author_id, title, slug, content, summary, status, view_count, created_at, published_at)
  FK: author_id → users(id)
- comments (id, article_id, user_id, parent_id, author_name, content, created_at)
  FK: article_id → articles(id), user_id → users(id), parent_id → comments(id)

Relationships:
- articles.author_id → users.id
- comments.article_id → articles.id
- comments.user_id → users.id
- comments.parent_id → comments.id (nested replies)
```

Estimated token cost: ~80 tokens for 3 tables. Acceptable for system prompt.
