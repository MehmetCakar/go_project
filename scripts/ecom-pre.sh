#!/usr/bin/env bash
set -euo pipefail
/usr/bin/psql -v ON_ERROR_STOP=1 -p 5433 -d ecom <<'SQL'
DO $$
BEGIN
  IF to_regclass('public.users') IS NOT NULL THEN
    IF NOT EXISTS (
      SELECT 1 FROM pg_constraint WHERE conname='uni_users_email'
    ) THEN
      ALTER TABLE public.users
        ADD CONSTRAINT uni_users_email UNIQUE (email);
    END IF;
  END IF;
END$$;
SQL
