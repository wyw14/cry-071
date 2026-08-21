BEGIN;

CREATE TABLE IF NOT EXISTS public_areas (
    id text PRIMARY KEY,
    name text NOT NULL,
    district text NOT NULL,
    active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS facility_categories (
    id text PRIMARY KEY,
    name text NOT NULL,
    description text NOT NULL DEFAULT '',
    active boolean NOT NULL DEFAULT true
);

CREATE TABLE IF NOT EXISTS feedback_subjects (
    code text PRIMARY KEY,
    name text NOT NULL,
    default_priority text NOT NULL CHECK (default_priority IN ('low','normal','high','critical')),
    default_assignee_id text NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS feedbacks (
    id text PRIMARY KEY,
    acceptance_number text NOT NULL UNIQUE,
    area_id text NOT NULL REFERENCES public_areas(id),
    subject_code text NOT NULL REFERENCES feedback_subjects(code),
    priority text NOT NULL CHECK (priority IN ('low','normal','high','critical')),
    status text NOT NULL CHECK (status IN ('pending_acceptance','needs_information','in_progress','pending_confirmation','closed','rejected')),
    assignee_id text NOT NULL DEFAULT '',
    version bigint NOT NULL CHECK (version > 0),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    due_at timestamptz NOT NULL,
    document jsonb NOT NULL
);

CREATE INDEX IF NOT EXISTS feedbacks_queue_idx ON feedbacks(area_id,status,due_at);
CREATE INDEX IF NOT EXISTS feedbacks_assignee_idx ON feedbacks(assignee_id,status,updated_at DESC);
CREATE INDEX IF NOT EXISTS feedbacks_subject_idx ON feedbacks(subject_code,created_at DESC);

CREATE TABLE IF NOT EXISTS timeline_events (
    id text PRIMARY KEY,
    feedback_id text NOT NULL REFERENCES feedbacks(id),
    sequence bigint NOT NULL CHECK (sequence > 0),
    visibility text NOT NULL CHECK (visibility IN ('public','internal')),
    occurred_at timestamptz NOT NULL,
    document jsonb NOT NULL,
    UNIQUE(feedback_id,sequence)
);

CREATE OR REPLACE FUNCTION reject_timeline_mutation() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'timeline events are immutable' USING ERRCODE='55000';
END $$;

DROP TRIGGER IF EXISTS timeline_no_update ON timeline_events;
CREATE TRIGGER timeline_no_update BEFORE UPDATE OR DELETE ON timeline_events
FOR EACH ROW EXECUTE FUNCTION reject_timeline_mutation();

CREATE TABLE IF NOT EXISTS replies (id text PRIMARY KEY,feedback_id text NOT NULL REFERENCES feedbacks(id),public boolean NOT NULL,created_at timestamptz NOT NULL,document jsonb NOT NULL);
CREATE TABLE IF NOT EXISTS supplements (id text PRIMARY KEY,feedback_id text NOT NULL REFERENCES feedbacks(id),created_at timestamptz NOT NULL,document jsonb NOT NULL);
CREATE TABLE IF NOT EXISTS satisfaction (feedback_id text PRIMARY KEY REFERENCES feedbacks(id),score integer NOT NULL CHECK(score BETWEEN 1 AND 5),created_at timestamptz NOT NULL,document jsonb NOT NULL);
CREATE TABLE IF NOT EXISTS attachments (id text PRIMARY KEY,feedback_id text NOT NULL REFERENCES feedbacks(id),created_at timestamptz NOT NULL,document jsonb NOT NULL);
CREATE TABLE IF NOT EXISTS merge_groups (id text PRIMARY KEY,primary_id text NOT NULL REFERENCES feedbacks(id),created_at timestamptz NOT NULL,document jsonb NOT NULL);
CREATE TABLE IF NOT EXISTS announcements (id text PRIMARY KEY,area_id text NOT NULL REFERENCES public_areas(id),published boolean NOT NULL DEFAULT false,created_at timestamptz NOT NULL,document jsonb NOT NULL);
CREATE TABLE IF NOT EXISTS audit_entries (id text PRIMARY KEY,resource text NOT NULL,resource_id text NOT NULL,occurred_at timestamptz NOT NULL,document jsonb NOT NULL);
CREATE INDEX IF NOT EXISTS audit_resource_idx ON audit_entries(resource,resource_id,occurred_at);
CREATE TABLE IF NOT EXISTS query_tokens (digest text PRIMARY KEY,feedback_id text NOT NULL REFERENCES feedbacks(id),expires_at timestamptz NOT NULL);
CREATE INDEX IF NOT EXISTS query_tokens_expiry_idx ON query_tokens(expires_at);
CREATE TABLE IF NOT EXISTS idempotency_keys (
    scope text NOT NULL,
    key text NOT NULL,
    request_hash text NOT NULL,
    resource_id text NOT NULL,
    expires_at timestamptz NOT NULL,
    PRIMARY KEY(scope,key)
);
CREATE INDEX IF NOT EXISTS idempotency_expiry_idx ON idempotency_keys(expires_at);

COMMIT;
