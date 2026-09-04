CREATE TABLE "users" (
  "id" uuid PRIMARY KEY DEFAULT (gen_random_uuid()),
  "name" varchar(255) NOT NULL,
  "email" varchar(255) UNIQUE NOT NULL,
  "password_hash" varchar(255) NOT NULL,
  "role" int NOT NULL DEFAULT 2001,
  "is_email_verified" boolean NOT NULL DEFAULT false,
  "email_verify_token_hash" varchar(64),
  "email_verify_token_expires_at" timestamptz,
  "password_reset_token_hash" varchar(64),
  "password_reset_token_expires_at" timestamptz,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "updated_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE TABLE "refresh_tokens" (
  "id" uuid PRIMARY KEY DEFAULT (gen_random_uuid()),
  "user_id" uuid NOT NULL,
  "token_hash" varchar(64) UNIQUE NOT NULL,
  "is_revoked" boolean NOT NULL DEFAULT false,
  "expires_at" timestamptz NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE TABLE "jobs" (
  "id" uuid PRIMARY KEY DEFAULT (gen_random_uuid()),
  "title" varchar(255) NOT NULL,
  "description" text NOT NULL,
  "company" varchar(255) NOT NULL,
  "location" varchar(255) NOT NULL,
  "salary_min" bigint,
  "salary_max" bigint,
  "created_by" uuid NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "updated_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE UNIQUE INDEX ON "users" ("email");

CREATE UNIQUE INDEX ON "refresh_tokens" ("token_hash");

CREATE INDEX ON "refresh_tokens" ("user_id");

CREATE INDEX ON "jobs" ("created_by");

CREATE INDEX ON "jobs" ("title");

COMMENT ON TABLE "users" IS 'Passwords and tokens are never stored in plaintext. Tokens stored as SHA-256 hashes.';

COMMENT ON COLUMN "users"."password_hash" IS 'bcrypt with secure cost factor';

COMMENT ON COLUMN "users"."role" IS '2001=User, 5150=Admin';

COMMENT ON COLUMN "users"."email_verify_token_hash" IS 'SHA-256 hash of one-time token';

COMMENT ON COLUMN "users"."password_reset_token_hash" IS 'SHA-256 hash of one-time token';

COMMENT ON TABLE "refresh_tokens" IS 'Supports token rotation + reuse detection. Reused tokens trigger full session revocation.';

COMMENT ON COLUMN "refresh_tokens"."token_hash" IS 'SHA-256 hash — raw token never persisted';

COMMENT ON COLUMN "refresh_tokens"."is_revoked" IS 'Set on reuse detection (ErrTokenReuse)';

COMMENT ON TABLE "jobs" IS 'Write operations (POST, DELETE) are scoped to Admin role (5150) only.';

COMMENT ON COLUMN "jobs"."created_by" IS 'Admin user who created the listing';

ALTER TABLE "refresh_tokens" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id") DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "jobs" ADD FOREIGN KEY ("created_by") REFERENCES "users" ("id") DEFERRABLE INITIALLY IMMEDIATE;
