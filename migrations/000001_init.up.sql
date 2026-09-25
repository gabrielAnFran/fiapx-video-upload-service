CREATE TABLE users (
  id UUID PRIMARY KEY,
  email TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE videos (
  id UUID PRIMARY KEY,
  user_id UUID NOT NULL REFERENCES users(id),
  original_filename TEXT NOT NULL,
  content_type TEXT NOT NULL,
  size_bytes BIGINT NOT NULL,
  source_bucket TEXT NOT NULL,
  source_object_key TEXT NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('UPLOADED','PROCESSING','COMPLETED','FAILED')),
  zip_bucket TEXT,
  zip_object_key TEXT,
  frame_count INT,
  error_message TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  completed_at TIMESTAMPTZ,
  failed_at TIMESTAMPTZ
);
CREATE INDEX idx_videos_user_status ON videos(user_id, status);
CREATE TABLE outbox (
  id BIGSERIAL PRIMARY KEY,
  event_id UUID NOT NULL UNIQUE,
  aggregate_id UUID NOT NULL,
  event_name TEXT NOT NULL,
  payload JSONB NOT NULL,
  headers JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  published_at TIMESTAMPTZ
);
CREATE INDEX idx_outbox_unpublished ON outbox(created_at) WHERE published_at IS NULL;
CREATE TABLE processed_events (
  event_id UUID PRIMARY KEY,
  processed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
