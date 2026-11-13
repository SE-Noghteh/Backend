-- Zedlancer PostgreSQL schema generated from supplied DBML.
-- Structural translation only. See DATABASE.md for application-enforced conditions.
-- Requires PostgreSQL with gen_random_uuid() available.
BEGIN;

CREATE TYPE "account_status" AS ENUM ('active', 'pending_admin_approval', 'rejected_pending_edit', 'pending_parent_approval', 'inactive');
CREATE TYPE "actor_type" AS ENUM ('seeker', 'parent', 'employer', 'admin');
CREATE TYPE "link_status" AS ENUM ('invited', 'active', 'revoked');
CREATE TYPE "opp_type" AS ENUM ('work', 'internship');
CREATE TYPE "opp_mode" AS ENUM ('online', 'onsite');
CREATE TYPE "opp_status" AS ENUM ('draft', 'in_review', 'published', 'closed', 'archived');
CREATE TYPE "screening_status" AS ENUM ('pending', 'approved', 'rejected', 'returned');
CREATE TYPE "application_status" AS ENUM ('sent', 'withdrawn', 'shortlisted', 'rejected', 'offered', 'accepted');
CREATE TYPE "offer_status" AS ENUM ('sent', 'accepted', 'declined', 'expired');
CREATE TYPE "agreement_version_status" AS ENUM ('proposed', 'active', 'superseded');
CREATE TYPE "consent_status" AS ENUM ('pending', 'granted', 'denied', 'withdrawn', 'expired');
CREATE TYPE "collab_status" AS ENUM ('active', 'paused', 'completed', 'cancelled', 'disputed');
CREATE TYPE "deliverable_status" AS ENUM ('submitted', 'revision_requested', 'approved');
CREATE TYPE "stop_type" AS ENUM ('pause', 'cancel', 'no_response', 'early_exit');
CREATE TYPE "finance_status" AS ENUM ('none', 'pending', 'success', 'failed', 'refunded', 'needs_review');
CREATE TYPE "skill_basis" AS ENUM ('self', 'used_in_collab', 'assessed');
CREATE TYPE "activity_mode" AS ENUM ('online', 'onsite', 'both');
CREATE TYPE "activity_acceptance" AS ENUM ('allowed_after_review', 'suspended', 'out_of_scope');
CREATE TYPE "review_status" AS ENUM ('hidden', 'published', 'removed');
CREATE TYPE "report_kind" AS ENUM ('safety', 'other');
CREATE TYPE "item_status" AS ENUM ('open', 'done', 'expired');

CREATE TABLE "users" (
  "account_status" account_status NOT NULL DEFAULT 'active',
  "deactivation_requested_at" timestamptz,
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "phone_number" varchar(20) NOT NULL,
  "password_hash" text NOT NULL,
  "phone_verified_at" timestamptz,
  "email" varchar(255),
  "email_verified_at" timestamptz,
  "terms_accepted_at" timestamptz,
  "privacy_accepted_at" timestamptz,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);

CREATE TABLE "addresses" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "province" varchar(100) NOT NULL,
  "city" varchar(100) NOT NULL,
  "postal_code" varchar(20) NOT NULL,
  "full_address" text NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);

CREATE TABLE "parents" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "user_id" uuid NOT NULL UNIQUE,
  "address_id" uuid,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);

CREATE TABLE "job_seekers" (
  "bio" text,
  "interests" text,
  "availability_note" varchar(255),
  "public_visibility" varchar(30),
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "user_id" uuid NOT NULL UNIQUE,
  "address_id" uuid,
  "parent_id" uuid,
  "first_name" varchar(100) NOT NULL,
  "last_name" varchar(100) NOT NULL,
  "national_id" varchar(20) NOT NULL,
  "gender" varchar(30),
  "birth_date" date NOT NULL,
  "parent_approval_status" varchar(20) NOT NULL,
  "parent_approved_at" timestamptz,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);

CREATE TABLE "employers" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "user_id" uuid NOT NULL UNIQUE,
  "employer_type" varchar(20) NOT NULL,
  "registration_status" varchar(30) NOT NULL,
  "verification_status" varchar(30) NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);

CREATE TABLE "individual_employers" (
  "employer_id" uuid NOT NULL,
  "address_id" uuid,
  "first_name" varchar(100) NOT NULL,
  "last_name" varchar(100) NOT NULL,
  "father_name" varchar(100),
  "national_id" varchar(20) NOT NULL,
  "birth_date" date,
  "brand_name" varchar(255),
  "website_url" text,
  "has_business_license" boolean NOT NULL DEFAULT false,
  PRIMARY KEY ("employer_id")
);

CREATE TABLE "legal_employers" (
  "employer_id" uuid NOT NULL,
  "address_id" uuid,
  "organization_name" varchar(255) NOT NULL,
  "legal_entity_type" varchar(50) NOT NULL,
  "national_legal_id" varchar(30) NOT NULL,
  "registration_number" varchar(50) NOT NULL,
  "economic_code" varchar(50),
  "representative_first_name" varchar(100) NOT NULL,
  "representative_last_name" varchar(100) NOT NULL,
  "representative_role" varchar(100),
  "representative_national_id" varchar(20) NOT NULL,
  "has_recent_registration_changes" boolean NOT NULL DEFAULT false,
  "website_url" text,
  PRIMARY KEY ("employer_id")
);

CREATE TABLE "activity_categories" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "name" varchar(150) NOT NULL,
  "description" text,
  "is_active" boolean NOT NULL DEFAULT true,
  PRIMARY KEY ("id")
);

CREATE TABLE "verification_documents" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "user_id" uuid NOT NULL,
  "document_type" varchar(60) NOT NULL,
  "file_key" text NOT NULL,
  "status" varchar(30) NOT NULL,
  "rejection_reason" text,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "reviewed_at" timestamptz,
  PRIMARY KEY ("id")
);

CREATE TABLE "parent_invitations" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "job_seeker_id" uuid NOT NULL,
  "code_hash" text NOT NULL,
  "status" varchar(20) NOT NULL,
  "expires_at" timestamptz NOT NULL,
  "used_at" timestamptz,
  "parent_id" uuid,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);

CREATE TABLE "parent_link_requests" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "job_seeker_id" uuid NOT NULL,
  "parent_id" uuid NOT NULL,
  "status" varchar(20) NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "responded_at" timestamptz,
  PRIMARY KEY ("id")
);

CREATE TABLE "skills" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "name" varchar(150) NOT NULL,
  "description" text,
  "is_active" boolean NOT NULL DEFAULT true,
  PRIMARY KEY ("id")
);

CREATE TABLE "job_seeker_skills" (
  "job_seeker_id" uuid NOT NULL,
  "skill_id" uuid NOT NULL,
  PRIMARY KEY ("job_seeker_id", "skill_id")
);

CREATE TABLE "job_postings" (
  "category_id" uuid,
  "activity_id" uuid,
  "opportunity_type" "opp_type" NOT NULL DEFAULT 'work',
  "work_mode" "opp_mode",
  "current_version_id" uuid,
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "employer_id" uuid NOT NULL,
  "title" varchar(255) NOT NULL,
  "description" text NOT NULL,
  "status" varchar(30) NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);

CREATE TABLE "job_posting_skills" (
  "job_posting_id" uuid NOT NULL,
  "skill_id" uuid NOT NULL,
  PRIMARY KEY ("job_posting_id", "skill_id")
);

CREATE TABLE "job_applications" (
  "motivation" text,
  "available_hours" integer,
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "job_posting_id" uuid NOT NULL,
  "job_seeker_id" uuid NOT NULL,
  "status" varchar(30) NOT NULL,
  "cover_letter" text,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);
CREATE UNIQUE INDEX "uq_job_applications_job_posting_id_job_seeker_id" ON "job_applications" ("job_posting_id", "job_seeker_id");

CREATE TABLE "conversations" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "job_application_id" uuid NOT NULL UNIQUE,
  "status" varchar(20) NOT NULL,
  "closed_reason" varchar(40),
  "closed_at" timestamptz,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);

CREATE TABLE "messages" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "conversation_id" uuid NOT NULL,
  "sender_user_id" uuid NOT NULL,
  "content" text,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "read_at" timestamptz,
  PRIMARY KEY ("id")
);

CREATE TABLE "message_attachments" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "message_id" uuid NOT NULL,
  "file_key" text NOT NULL,
  "file_name" text NOT NULL,
  "mime_type" varchar(150) NOT NULL,
  "file_size" bigint NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);

CREATE TABLE "work_contracts" (
  "current_version_id" uuid,
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "job_application_id" uuid NOT NULL UNIQUE,
  "title" varchar(255) NOT NULL,
  "description" text NOT NULL,
  "status" varchar(30) NOT NULL,
  "total_amount" bigint NOT NULL,
  "funding_status" varchar(30) NOT NULL,
  "funded_at" timestamptz,
  "start_date" timestamptz,
  "end_date" timestamptz,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);

CREATE TABLE "contract_approvals" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "work_contract_id" uuid NOT NULL,
  "user_id" uuid NOT NULL,
  "role" varchar(20) NOT NULL,
  "status" varchar(20) NOT NULL,
  "contract_version" integer NOT NULL DEFAULT 1,
  "approved_at" timestamptz,
  "rejected_at" timestamptz,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);
CREATE UNIQUE INDEX "uq_contract_approvals_work_contract_id_user_id_contract_version" ON "contract_approvals" ("work_contract_id", "user_id", "contract_version");

CREATE TABLE "contract_milestones" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "work_contract_id" uuid NOT NULL,
  "title" varchar(255) NOT NULL,
  "description" text,
  "amount" bigint NOT NULL,
  "due_date" timestamptz,
  "status" varchar(30) NOT NULL,
  PRIMARY KEY ("id")
);

CREATE TABLE "payments" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "work_contract_id" uuid NOT NULL,
  "amount" bigint NOT NULL,
  "status" varchar(30) NOT NULL,
  "gateway_reference" text,
  "idempotency_key" varchar(100) NOT NULL,
  "paid_at" timestamptz,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);

CREATE TABLE "payouts" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "work_contract_id" uuid NOT NULL,
  "job_seeker_id" uuid NOT NULL,
  "amount" bigint NOT NULL,
  "status" varchar(30) NOT NULL,
  "idempotency_key" varchar(100) NOT NULL,
  "paid_at" timestamptz,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);

CREATE TABLE "financial_transactions" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "work_contract_id" uuid NOT NULL,
  "type" varchar(30) NOT NULL,
  "amount" bigint NOT NULL,
  "reference_id" uuid,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);

CREATE TABLE "work_submissions" (
  "collaboration_id" uuid,
  "version_no" integer,
  "approved_at" timestamptz,
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "work_contract_id" uuid NOT NULL,
  "description" text NOT NULL,
  "status" varchar(30) NOT NULL,
  "review_note" text,
  "submitted_at" timestamptz NOT NULL,
  PRIMARY KEY ("id")
);

CREATE TABLE "disputes" (
  "collaboration_id" uuid,
  "respond_by" timestamptz,
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "work_contract_id" uuid NOT NULL,
  "created_by_user_id" uuid NOT NULL,
  "assigned_support_user_id" uuid,
  "reason" text NOT NULL,
  "status" varchar(30) NOT NULL,
  "resolution" varchar(30),
  "resolution_note" text,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "resolved_at" timestamptz,
  PRIMARY KEY ("id")
);

CREATE TABLE "dispute_evidence" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "dispute_id" uuid NOT NULL,
  "uploaded_by_user_id" uuid NOT NULL,
  "file_key" text NOT NULL,
  "file_name" text NOT NULL,
  "mime_type" varchar(150) NOT NULL,
  "file_size" bigint NOT NULL,
  "description" text,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);

CREATE TABLE "parent_links" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "parent_id" uuid NOT NULL,
  "job_seeker_id" uuid NOT NULL,
  "status" "link_status" NOT NULL,
  "invited_at" timestamptz,
  "accepted_at" timestamptz,
  "revoked_at" timestamptz,
  PRIMARY KEY ("id")
);
CREATE UNIQUE INDEX "uq_parent_links_parent_id_job_seeker_id" ON "parent_links" ("parent_id", "job_seeker_id");

CREATE TABLE "admin_actions" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "admin_user_id" uuid NOT NULL,
  "target_type" varchar NOT NULL,
  "target_id" uuid NOT NULL,
  "action" varchar NOT NULL,
  "reason" text NOT NULL,
  "changes" jsonb,
  "created_at" timestamptz DEFAULT now(),
  PRIMARY KEY ("id")
);

CREATE TABLE "categories" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "parent_id" uuid,
  "title" varchar NOT NULL,
  "active" boolean DEFAULT true,
  PRIMARY KEY ("id")
);

CREATE TABLE "activities" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "category_id" uuid NOT NULL,
  "title" varchar NOT NULL UNIQUE,
  "allowed_modes" "activity_mode" NOT NULL,
  "min_requirements" text,
  "supervision_requirements" text,
  "acceptance" "activity_acceptance" NOT NULL DEFAULT 'allowed_after_review',
  "active" boolean DEFAULT true,
  PRIMARY KEY ("id")
);

CREATE TABLE "files" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "owner_type" "actor_type" NOT NULL,
  "owner_id" uuid NOT NULL,
  "mime" varchar NOT NULL,
  "size_bytes" bigint NOT NULL,
  "storage_key" varchar NOT NULL,
  "created_at" timestamptz,
  PRIMARY KEY ("id")
);

CREATE TABLE "portfolio_items" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "job_seeker_id" uuid NOT NULL,
  "title" varchar,
  "description" text,
  "file_id" uuid,
  "visibility" varchar NOT NULL,
  PRIMARY KEY ("id")
);

CREATE TABLE "skill_evidence" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "job_seeker_id" uuid NOT NULL,
  "skill_id" uuid NOT NULL,
  "collaboration_id" uuid,
  "portfolio_item_id" uuid,
  "challenge_attempt_id" uuid,
  "mentor_summary" text,
  PRIMARY KEY ("id")
);

CREATE TABLE "challenges" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "skill_id" uuid NOT NULL,
  "level" varchar,
  "description" text,
  "input_file_id" uuid,
  "expected_output" text,
  "criteria" text,
  "max_attempts" int,
  PRIMARY KEY ("id")
);

CREATE TABLE "challenge_attempts" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "challenge_id" uuid NOT NULL,
  "job_seeker_id" uuid NOT NULL,
  "answer_text" text,
  "answer_file_id" uuid,
  "eval_method" varchar,
  "result" varchar,
  "feedback" text,
  PRIMARY KEY ("id")
);

CREATE TABLE "growth_goals" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "job_seeker_id" uuid NOT NULL,
  "skill_id" uuid NOT NULL,
  PRIMARY KEY ("id")
);

CREATE TABLE "availability_slots" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "job_seeker_id" uuid NOT NULL,
  "kind" varchar,
  "from_at" timestamptz,
  "to_at" timestamptz,
  PRIMARY KEY ("id")
);

CREATE TABLE "job_posting_versions" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "job_posting_id" uuid NOT NULL,
  "version_no" int NOT NULL,
  "title" varchar NOT NULL,
  "description" text NOT NULL,
  "eligibility" text,
  "collab_style" varchar,
  "duration" varchar,
  "expected_hours" int,
  "capacity" int NOT NULL DEFAULT 1,
  "apply_deadline" timestamptz,
  "delivery_deadline" timestamptz,
  "expected_output" text,
  "amount" decimal(14,2),
  "pay_formula" varchar,
  "pay_status" varchar,
  "max_revisions" int,
  "location_info" jsonb,
  "created_at" timestamptz,
  PRIMARY KEY ("id")
);
CREATE UNIQUE INDEX "uq_job_posting_versions_job_posting_id_version_no" ON "job_posting_versions" ("job_posting_id", "version_no");

CREATE TABLE "job_posting_version_skills" (
  "job_posting_version_id" uuid NOT NULL,
  "skill_id" uuid NOT NULL,
  PRIMARY KEY ("job_posting_version_id", "skill_id")
);

CREATE TABLE "internship_details" (
  "job_posting_version_id" uuid,
  "learning_goal" text NOT NULL,
  "practice_skills" text,
  "activities" text,
  "prerequisites" text,
  "start_date" date,
  "end_date" date,
  "expected_hours" int,
  "mentor_name" varchar NOT NULL,
  "mentor_contact" varchar,
  "stipend" decimal(14,2),
  "completion_criteria" text NOT NULL,
  PRIMARY KEY ("job_posting_version_id")
);

CREATE TABLE "screening_results" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "job_posting_version_id" uuid NOT NULL,
  "rule_code" varchar NOT NULL,
  "flagged_text" text,
  "reason" text,
  "status" "screening_status" NOT NULL DEFAULT 'pending',
  "decided_by_user_id" uuid,
  "decision_reason" text,
  "decided_at" timestamptz,
  PRIMARY KEY ("id")
);

CREATE TABLE "saved_job_postings" (
  "job_seeker_id" uuid NOT NULL,
  "job_posting_id" uuid NOT NULL,
  PRIMARY KEY ("job_seeker_id", "job_posting_id")
);

CREATE TABLE "offers" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "job_application_id" uuid NOT NULL,
  "expires_at" timestamptz NOT NULL,
  "status" "offer_status" NOT NULL DEFAULT 'sent',
  "created_at" timestamptz,
  PRIMARY KEY ("id")
);

CREATE TABLE "contract_versions" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "work_contract_id" uuid NOT NULL,
  "version_no" int NOT NULL,
  "terms" jsonb NOT NULL,
  "status" "agreement_version_status" NOT NULL,
  "created_by_type" "actor_type",
  "created_by_user_id" uuid,
  "created_at" timestamptz,
  PRIMARY KEY ("id")
);
CREATE UNIQUE INDEX "uq_contract_versions_work_contract_id_version_no" ON "contract_versions" ("work_contract_id", "version_no");

CREATE TABLE "contract_version_acceptances" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "contract_version_id" uuid NOT NULL,
  "acceptor_type" "actor_type" NOT NULL,
  "acceptor_user_id" uuid NOT NULL,
  "accepted_at" timestamptz NOT NULL,
  PRIMARY KEY ("id")
);
CREATE UNIQUE INDEX "uq_contract_version_acceptances_contract_version_id_acceptor_type_acceptor_user_id" ON "contract_version_acceptances" ("contract_version_id", "acceptor_type", "acceptor_user_id");

CREATE TABLE "contract_parent_consents" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "contract_version_id" uuid NOT NULL,
  "parent_id" uuid NOT NULL,
  "job_seeker_id" uuid NOT NULL,
  "status" "consent_status" NOT NULL DEFAULT 'pending',
  "deny_reason" text,
  "respond_by" timestamptz,
  "decided_at" timestamptz,
  "withdrawn_at" timestamptz,
  PRIMARY KEY ("id")
);

CREATE TABLE "collaborations" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "work_contract_id" uuid NOT NULL UNIQUE,
  "employer_id" uuid NOT NULL,
  "job_seeker_id" uuid NOT NULL,
  "active_contract_version_id" uuid NOT NULL,
  "status" "collab_status" NOT NULL DEFAULT 'active',
  "started_at" timestamptz,
  "ended_at" timestamptz,
  "end_outcome" varchar,
  PRIMARY KEY ("id")
);

CREATE TABLE "tasks" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "collaboration_id" uuid NOT NULL,
  "assignee_type" "actor_type",
  "assignee_user_id" uuid,
  "title" varchar NOT NULL,
  "description" text,
  "status" varchar,
  "due_at" timestamptz,
  PRIMARY KEY ("id")
);

CREATE TABLE "collaboration_message_files" (
  "collaboration_message_id" uuid NOT NULL,
  "file_id" uuid NOT NULL,
  PRIMARY KEY ("collaboration_message_id", "file_id")
);

CREATE TABLE "deliverables" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "collaboration_id" uuid NOT NULL,
  "version_no" int NOT NULL,
  "note" text,
  "status" "deliverable_status" NOT NULL DEFAULT 'submitted',
  "submitted_at" timestamptz,
  "approved_at" timestamptz,
  PRIMARY KEY ("id")
);
CREATE UNIQUE INDEX "uq_deliverables_collaboration_id_version_no" ON "deliverables" ("collaboration_id", "version_no");

CREATE TABLE "deliverable_files" (
  "deliverable_id" uuid NOT NULL,
  "file_id" uuid NOT NULL,
  PRIMARY KEY ("deliverable_id", "file_id")
);

CREATE TABLE "revision_requests" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "deliverable_id" uuid NOT NULL,
  "description" text NOT NULL,
  "annotation" jsonb,
  "status" varchar,
  "created_at" timestamptz,
  PRIMARY KEY ("id")
);

CREATE TABLE "stop_requests" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "collaboration_id" uuid NOT NULL,
  "requested_by_type" "actor_type" NOT NULL,
  "requested_by_user_id" uuid NOT NULL,
  "type" "stop_type" NOT NULL,
  "reason" text,
  "status" varchar,
  "created_at" timestamptz,
  PRIMARY KEY ("id")
);

CREATE TABLE "extension_requests" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "collaboration_id" uuid NOT NULL,
  "new_contract_version_id" uuid NOT NULL,
  PRIMARY KEY ("id")
);

CREATE TABLE "collab_events" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "collaboration_id" uuid NOT NULL,
  "actor_type" "actor_type",
  "actor_user_id" uuid,
  "event_type" varchar NOT NULL,
  "payload" jsonb,
  "created_at" timestamptz DEFAULT now(),
  PRIMARY KEY ("id")
);

CREATE TABLE "dispute_decisions" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "dispute_id" uuid NOT NULL,
  "admin_user_id" uuid NOT NULL,
  "decision" text NOT NULL,
  "reason" text NOT NULL,
  "collab_effect" varchar,
  "financial_effect" varchar,
  "created_at" timestamptz,
  PRIMARY KEY ("id")
);

CREATE TABLE "reports" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "reporter_type" "actor_type" NOT NULL,
  "reporter_user_id" uuid NOT NULL,
  "target_type" varchar NOT NULL,
  "target_id" uuid NOT NULL,
  "kind" "report_kind" NOT NULL,
  "reason" text,
  "status" varchar,
  "priority" varchar,
  "created_at" timestamptz,
  PRIMARY KEY ("id")
);

CREATE TABLE "report_actions" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "report_id" uuid NOT NULL,
  "admin_user_id" uuid NOT NULL,
  "action" varchar NOT NULL,
  "reason" text,
  "created_at" timestamptz,
  PRIMARY KEY ("id")
);

CREATE TABLE "appeals" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "appellant_type" "actor_type" NOT NULL,
  "appellant_user_id" uuid NOT NULL,
  "target_type" varchar NOT NULL,
  "target_id" uuid NOT NULL,
  "reason" text,
  "status" varchar,
  "reviewer_user_id" uuid,
  "result" text,
  PRIMARY KEY ("id")
);

CREATE TABLE "reviews" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "collaboration_id" uuid NOT NULL,
  "author_type" "actor_type" NOT NULL,
  "author_user_id" uuid NOT NULL,
  "subject_type" "actor_type" NOT NULL,
  "subject_user_id" uuid NOT NULL,
  "rating" int,
  "criteria" jsonb,
  "comment" text,
  "status" "review_status" NOT NULL DEFAULT 'hidden',
  PRIMARY KEY ("id")
);
CREATE UNIQUE INDEX "uq_reviews_collaboration_id_author_type_author_user_id" ON "reviews" ("collaboration_id", "author_type", "author_user_id");

CREATE TABLE "notifications" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "recipient_type" "actor_type" NOT NULL,
  "recipient_user_id" uuid NOT NULL,
  "type" varchar NOT NULL,
  "link_type" varchar,
  "link_id" uuid,
  "read_at" timestamptz,
  "created_at" timestamptz,
  PRIMARY KEY ("id")
);

CREATE TABLE "action_items" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "owner_type" "actor_type" NOT NULL,
  "owner_user_id" uuid NOT NULL,
  "kind" varchar NOT NULL,
  "ref_type" varchar,
  "ref_id" uuid,
  "status" "item_status" NOT NULL DEFAULT 'open',
  "due_at" timestamptz,
  PRIMARY KEY ("id")
);

CREATE TABLE "user_sessions" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "user_id" uuid NOT NULL,
  "token_hash" text NOT NULL,
  "device_info" text,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "expires_at" timestamptz NOT NULL,
  "revoked_at" timestamptz,
  PRIMARY KEY ("id")
);

CREATE TABLE "admin_profiles" (
  "user_id" uuid NOT NULL,
  "admin_scope" varchar(100) NOT NULL,
  "is_active" boolean NOT NULL DEFAULT true,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("user_id")
);

CREATE TABLE "collaboration_messages" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "collaboration_id" uuid NOT NULL,
  "sender_user_id" uuid NOT NULL,
  "body" text,
  "hidden" boolean NOT NULL DEFAULT false,
  "read_at" timestamptz,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);

CREATE TABLE "financial_audit_events" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "financial_transaction_id" uuid NOT NULL,
  "event_type" varchar(60) NOT NULL,
  "decision_ref" uuid,
  "idempotency_key" varchar(100) NOT NULL UNIQUE,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);

ALTER TABLE "parents" ADD CONSTRAINT "fk_1_parents_users" FOREIGN KEY ("user_id") REFERENCES "users" ("id");
ALTER TABLE "parents" ADD CONSTRAINT "fk_2_parents_addresses" FOREIGN KEY ("address_id") REFERENCES "addresses" ("id");
ALTER TABLE "job_seekers" ADD CONSTRAINT "fk_3_job_seekers_users" FOREIGN KEY ("user_id") REFERENCES "users" ("id");
ALTER TABLE "job_seekers" ADD CONSTRAINT "fk_4_job_seekers_addresses" FOREIGN KEY ("address_id") REFERENCES "addresses" ("id");
ALTER TABLE "job_seekers" ADD CONSTRAINT "fk_5_job_seekers_parents" FOREIGN KEY ("parent_id") REFERENCES "parents" ("id");
ALTER TABLE "employers" ADD CONSTRAINT "fk_6_employers_users" FOREIGN KEY ("user_id") REFERENCES "users" ("id");
ALTER TABLE "individual_employers" ADD CONSTRAINT "fk_7_individual_employers_employers" FOREIGN KEY ("employer_id") REFERENCES "employers" ("id");
ALTER TABLE "individual_employers" ADD CONSTRAINT "fk_8_individual_employers_addresses" FOREIGN KEY ("address_id") REFERENCES "addresses" ("id");
ALTER TABLE "legal_employers" ADD CONSTRAINT "fk_9_legal_employers_employers" FOREIGN KEY ("employer_id") REFERENCES "employers" ("id");
ALTER TABLE "legal_employers" ADD CONSTRAINT "fk_10_legal_employers_addresses" FOREIGN KEY ("address_id") REFERENCES "addresses" ("id");
ALTER TABLE "verification_documents" ADD CONSTRAINT "fk_11_verification_documents_users" FOREIGN KEY ("user_id") REFERENCES "users" ("id");
ALTER TABLE "parent_invitations" ADD CONSTRAINT "fk_12_parent_invitations_job_seekers" FOREIGN KEY ("job_seeker_id") REFERENCES "job_seekers" ("id");
ALTER TABLE "parent_invitations" ADD CONSTRAINT "fk_13_parent_invitations_parents" FOREIGN KEY ("parent_id") REFERENCES "parents" ("id");
ALTER TABLE "parent_link_requests" ADD CONSTRAINT "fk_14_parent_link_requests_job_seekers" FOREIGN KEY ("job_seeker_id") REFERENCES "job_seekers" ("id");
ALTER TABLE "parent_link_requests" ADD CONSTRAINT "fk_15_parent_link_requests_parents" FOREIGN KEY ("parent_id") REFERENCES "parents" ("id");
ALTER TABLE "job_seeker_skills" ADD CONSTRAINT "fk_16_job_seeker_skills_job_seekers" FOREIGN KEY ("job_seeker_id") REFERENCES "job_seekers" ("id");
ALTER TABLE "job_seeker_skills" ADD CONSTRAINT "fk_17_job_seeker_skills_skills" FOREIGN KEY ("skill_id") REFERENCES "skills" ("id");
ALTER TABLE "job_postings" ADD CONSTRAINT "fk_18_job_postings_employers" FOREIGN KEY ("employer_id") REFERENCES "employers" ("id");
ALTER TABLE "job_posting_skills" ADD CONSTRAINT "fk_19_job_posting_skills_job_postings" FOREIGN KEY ("job_posting_id") REFERENCES "job_postings" ("id");
ALTER TABLE "job_posting_skills" ADD CONSTRAINT "fk_20_job_posting_skills_skills" FOREIGN KEY ("skill_id") REFERENCES "skills" ("id");
ALTER TABLE "job_applications" ADD CONSTRAINT "fk_21_job_applications_job_postings" FOREIGN KEY ("job_posting_id") REFERENCES "job_postings" ("id");
ALTER TABLE "job_applications" ADD CONSTRAINT "fk_22_job_applications_job_seekers" FOREIGN KEY ("job_seeker_id") REFERENCES "job_seekers" ("id");
ALTER TABLE "conversations" ADD CONSTRAINT "fk_23_conversations_job_applications" FOREIGN KEY ("job_application_id") REFERENCES "job_applications" ("id");
ALTER TABLE "messages" ADD CONSTRAINT "fk_24_messages_conversations" FOREIGN KEY ("conversation_id") REFERENCES "conversations" ("id");
ALTER TABLE "messages" ADD CONSTRAINT "fk_25_messages_users" FOREIGN KEY ("sender_user_id") REFERENCES "users" ("id");
ALTER TABLE "message_attachments" ADD CONSTRAINT "fk_26_message_attachments_messages" FOREIGN KEY ("message_id") REFERENCES "messages" ("id");
ALTER TABLE "work_contracts" ADD CONSTRAINT "fk_27_work_contracts_job_applications" FOREIGN KEY ("job_application_id") REFERENCES "job_applications" ("id");
ALTER TABLE "contract_approvals" ADD CONSTRAINT "fk_28_contract_approvals_work_contracts" FOREIGN KEY ("work_contract_id") REFERENCES "work_contracts" ("id");
ALTER TABLE "contract_approvals" ADD CONSTRAINT "fk_29_contract_approvals_users" FOREIGN KEY ("user_id") REFERENCES "users" ("id");
ALTER TABLE "contract_milestones" ADD CONSTRAINT "fk_30_contract_milestones_work_contracts" FOREIGN KEY ("work_contract_id") REFERENCES "work_contracts" ("id");
ALTER TABLE "payments" ADD CONSTRAINT "fk_31_payments_work_contracts" FOREIGN KEY ("work_contract_id") REFERENCES "work_contracts" ("id");
ALTER TABLE "payouts" ADD CONSTRAINT "fk_32_payouts_work_contracts" FOREIGN KEY ("work_contract_id") REFERENCES "work_contracts" ("id");
ALTER TABLE "payouts" ADD CONSTRAINT "fk_33_payouts_job_seekers" FOREIGN KEY ("job_seeker_id") REFERENCES "job_seekers" ("id");
ALTER TABLE "financial_transactions" ADD CONSTRAINT "fk_34_financial_transactions_work_contracts" FOREIGN KEY ("work_contract_id") REFERENCES "work_contracts" ("id");
ALTER TABLE "work_submissions" ADD CONSTRAINT "fk_35_work_submissions_work_contracts" FOREIGN KEY ("work_contract_id") REFERENCES "work_contracts" ("id");
ALTER TABLE "disputes" ADD CONSTRAINT "fk_36_disputes_work_contracts" FOREIGN KEY ("work_contract_id") REFERENCES "work_contracts" ("id");
ALTER TABLE "disputes" ADD CONSTRAINT "fk_37_disputes_users" FOREIGN KEY ("created_by_user_id") REFERENCES "users" ("id");
ALTER TABLE "disputes" ADD CONSTRAINT "fk_38_disputes_users" FOREIGN KEY ("assigned_support_user_id") REFERENCES "users" ("id");
ALTER TABLE "dispute_evidence" ADD CONSTRAINT "fk_39_dispute_evidence_disputes" FOREIGN KEY ("dispute_id") REFERENCES "disputes" ("id");
ALTER TABLE "dispute_evidence" ADD CONSTRAINT "fk_40_dispute_evidence_users" FOREIGN KEY ("uploaded_by_user_id") REFERENCES "users" ("id");
ALTER TABLE "user_sessions" ADD CONSTRAINT "fk_41_user_sessions_users" FOREIGN KEY ("user_id") REFERENCES "users" ("id");
ALTER TABLE "admin_profiles" ADD CONSTRAINT "fk_42_admin_profiles_users" FOREIGN KEY ("user_id") REFERENCES "users" ("id");
ALTER TABLE "parent_links" ADD CONSTRAINT "fk_43_parent_links_parents" FOREIGN KEY ("parent_id") REFERENCES "parents" ("id");
ALTER TABLE "parent_links" ADD CONSTRAINT "fk_44_parent_links_job_seekers" FOREIGN KEY ("job_seeker_id") REFERENCES "job_seekers" ("id");
ALTER TABLE "admin_actions" ADD CONSTRAINT "fk_45_admin_actions_users" FOREIGN KEY ("admin_user_id") REFERENCES "users" ("id");
ALTER TABLE "categories" ADD CONSTRAINT "fk_46_categories_categories" FOREIGN KEY ("parent_id") REFERENCES "categories" ("id");
ALTER TABLE "activities" ADD CONSTRAINT "fk_47_activities_categories" FOREIGN KEY ("category_id") REFERENCES "categories" ("id");
ALTER TABLE "portfolio_items" ADD CONSTRAINT "fk_48_portfolio_items_job_seekers" FOREIGN KEY ("job_seeker_id") REFERENCES "job_seekers" ("id");
ALTER TABLE "portfolio_items" ADD CONSTRAINT "fk_49_portfolio_items_files" FOREIGN KEY ("file_id") REFERENCES "files" ("id");
ALTER TABLE "skill_evidence" ADD CONSTRAINT "fk_50_skill_evidence_job_seeker_skills" FOREIGN KEY ("job_seeker_id", "skill_id") REFERENCES "job_seeker_skills" ("job_seeker_id", "skill_id");
ALTER TABLE "skill_evidence" ADD CONSTRAINT "fk_51_skill_evidence_collaborations" FOREIGN KEY ("collaboration_id") REFERENCES "collaborations" ("id");
ALTER TABLE "skill_evidence" ADD CONSTRAINT "fk_52_skill_evidence_portfolio_items" FOREIGN KEY ("portfolio_item_id") REFERENCES "portfolio_items" ("id");
ALTER TABLE "skill_evidence" ADD CONSTRAINT "fk_53_skill_evidence_challenge_attempts" FOREIGN KEY ("challenge_attempt_id") REFERENCES "challenge_attempts" ("id");
ALTER TABLE "challenges" ADD CONSTRAINT "fk_54_challenges_skills" FOREIGN KEY ("skill_id") REFERENCES "skills" ("id");
ALTER TABLE "challenges" ADD CONSTRAINT "fk_55_challenges_files" FOREIGN KEY ("input_file_id") REFERENCES "files" ("id");
ALTER TABLE "challenge_attempts" ADD CONSTRAINT "fk_56_challenge_attempts_challenges" FOREIGN KEY ("challenge_id") REFERENCES "challenges" ("id");
ALTER TABLE "challenge_attempts" ADD CONSTRAINT "fk_57_challenge_attempts_job_seekers" FOREIGN KEY ("job_seeker_id") REFERENCES "job_seekers" ("id");
ALTER TABLE "challenge_attempts" ADD CONSTRAINT "fk_58_challenge_attempts_files" FOREIGN KEY ("answer_file_id") REFERENCES "files" ("id");
ALTER TABLE "growth_goals" ADD CONSTRAINT "fk_59_growth_goals_job_seekers" FOREIGN KEY ("job_seeker_id") REFERENCES "job_seekers" ("id");
ALTER TABLE "growth_goals" ADD CONSTRAINT "fk_60_growth_goals_skills" FOREIGN KEY ("skill_id") REFERENCES "skills" ("id");
ALTER TABLE "availability_slots" ADD CONSTRAINT "fk_61_availability_slots_job_seekers" FOREIGN KEY ("job_seeker_id") REFERENCES "job_seekers" ("id");
ALTER TABLE "job_postings" ADD CONSTRAINT "fk_62_job_postings_categories" FOREIGN KEY ("category_id") REFERENCES "categories" ("id");
ALTER TABLE "job_postings" ADD CONSTRAINT "fk_63_job_postings_activities" FOREIGN KEY ("activity_id") REFERENCES "activities" ("id");
ALTER TABLE "job_postings" ADD CONSTRAINT "fk_64_job_postings_job_posting_versions" FOREIGN KEY ("current_version_id") REFERENCES "job_posting_versions" ("id");
ALTER TABLE "job_posting_versions" ADD CONSTRAINT "fk_65_job_posting_versions_job_postings" FOREIGN KEY ("job_posting_id") REFERENCES "job_postings" ("id");
ALTER TABLE "job_posting_version_skills" ADD CONSTRAINT "fk_66_job_posting_version_skills_job_posting_versions" FOREIGN KEY ("job_posting_version_id") REFERENCES "job_posting_versions" ("id");
ALTER TABLE "job_posting_version_skills" ADD CONSTRAINT "fk_67_job_posting_version_skills_skills" FOREIGN KEY ("skill_id") REFERENCES "skills" ("id");
ALTER TABLE "internship_details" ADD CONSTRAINT "fk_68_internship_details_job_posting_versions" FOREIGN KEY ("job_posting_version_id") REFERENCES "job_posting_versions" ("id");
ALTER TABLE "screening_results" ADD CONSTRAINT "fk_69_screening_results_job_posting_versions" FOREIGN KEY ("job_posting_version_id") REFERENCES "job_posting_versions" ("id");
ALTER TABLE "screening_results" ADD CONSTRAINT "fk_70_screening_results_users" FOREIGN KEY ("decided_by_user_id") REFERENCES "users" ("id");
ALTER TABLE "saved_job_postings" ADD CONSTRAINT "fk_71_saved_job_postings_job_seekers" FOREIGN KEY ("job_seeker_id") REFERENCES "job_seekers" ("id");
ALTER TABLE "saved_job_postings" ADD CONSTRAINT "fk_72_saved_job_postings_job_postings" FOREIGN KEY ("job_posting_id") REFERENCES "job_postings" ("id");
ALTER TABLE "offers" ADD CONSTRAINT "fk_73_offers_job_applications" FOREIGN KEY ("job_application_id") REFERENCES "job_applications" ("id");
ALTER TABLE "contract_versions" ADD CONSTRAINT "fk_74_contract_versions_work_contracts" FOREIGN KEY ("work_contract_id") REFERENCES "work_contracts" ("id");
ALTER TABLE "work_contracts" ADD CONSTRAINT "fk_75_work_contracts_contract_versions" FOREIGN KEY ("current_version_id") REFERENCES "contract_versions" ("id");
ALTER TABLE "contract_versions" ADD CONSTRAINT "fk_76_contract_versions_users" FOREIGN KEY ("created_by_user_id") REFERENCES "users" ("id");
ALTER TABLE "contract_version_acceptances" ADD CONSTRAINT "fk_77_contract_version_acceptances_contract_versions" FOREIGN KEY ("contract_version_id") REFERENCES "contract_versions" ("id");
ALTER TABLE "contract_version_acceptances" ADD CONSTRAINT "fk_78_contract_version_acceptances_users" FOREIGN KEY ("acceptor_user_id") REFERENCES "users" ("id");
ALTER TABLE "contract_parent_consents" ADD CONSTRAINT "fk_79_contract_parent_consents_contract_versions" FOREIGN KEY ("contract_version_id") REFERENCES "contract_versions" ("id");
ALTER TABLE "contract_parent_consents" ADD CONSTRAINT "fk_80_contract_parent_consents_parents" FOREIGN KEY ("parent_id") REFERENCES "parents" ("id");
ALTER TABLE "contract_parent_consents" ADD CONSTRAINT "fk_81_contract_parent_consents_job_seekers" FOREIGN KEY ("job_seeker_id") REFERENCES "job_seekers" ("id");
ALTER TABLE "collaborations" ADD CONSTRAINT "fk_82_collaborations_work_contracts" FOREIGN KEY ("work_contract_id") REFERENCES "work_contracts" ("id");
ALTER TABLE "collaborations" ADD CONSTRAINT "fk_83_collaborations_employers" FOREIGN KEY ("employer_id") REFERENCES "employers" ("id");
ALTER TABLE "collaborations" ADD CONSTRAINT "fk_84_collaborations_job_seekers" FOREIGN KEY ("job_seeker_id") REFERENCES "job_seekers" ("id");
ALTER TABLE "collaborations" ADD CONSTRAINT "fk_85_collaborations_contract_versions" FOREIGN KEY ("active_contract_version_id") REFERENCES "contract_versions" ("id");
ALTER TABLE "tasks" ADD CONSTRAINT "fk_86_tasks_collaborations" FOREIGN KEY ("collaboration_id") REFERENCES "collaborations" ("id");
ALTER TABLE "collaboration_messages" ADD CONSTRAINT "fk_87_collaboration_messages_collaborations" FOREIGN KEY ("collaboration_id") REFERENCES "collaborations" ("id");
ALTER TABLE "collaboration_messages" ADD CONSTRAINT "fk_88_collaboration_messages_users" FOREIGN KEY ("sender_user_id") REFERENCES "users" ("id");
ALTER TABLE "collaboration_message_files" ADD CONSTRAINT "fk_89_collaboration_message_files_collaboration_messages" FOREIGN KEY ("collaboration_message_id") REFERENCES "collaboration_messages" ("id");
ALTER TABLE "collaboration_message_files" ADD CONSTRAINT "fk_90_collaboration_message_files_files" FOREIGN KEY ("file_id") REFERENCES "files" ("id");
ALTER TABLE "deliverables" ADD CONSTRAINT "fk_91_deliverables_collaborations" FOREIGN KEY ("collaboration_id") REFERENCES "collaborations" ("id");
ALTER TABLE "deliverable_files" ADD CONSTRAINT "fk_92_deliverable_files_deliverables" FOREIGN KEY ("deliverable_id") REFERENCES "deliverables" ("id");
ALTER TABLE "deliverable_files" ADD CONSTRAINT "fk_93_deliverable_files_files" FOREIGN KEY ("file_id") REFERENCES "files" ("id");
ALTER TABLE "revision_requests" ADD CONSTRAINT "fk_94_revision_requests_deliverables" FOREIGN KEY ("deliverable_id") REFERENCES "deliverables" ("id");
ALTER TABLE "stop_requests" ADD CONSTRAINT "fk_95_stop_requests_collaborations" FOREIGN KEY ("collaboration_id") REFERENCES "collaborations" ("id");
ALTER TABLE "stop_requests" ADD CONSTRAINT "fk_96_stop_requests_users" FOREIGN KEY ("requested_by_user_id") REFERENCES "users" ("id");
ALTER TABLE "extension_requests" ADD CONSTRAINT "fk_97_extension_requests_collaborations" FOREIGN KEY ("collaboration_id") REFERENCES "collaborations" ("id");
ALTER TABLE "extension_requests" ADD CONSTRAINT "fk_98_extension_requests_contract_versions" FOREIGN KEY ("new_contract_version_id") REFERENCES "contract_versions" ("id");
ALTER TABLE "collab_events" ADD CONSTRAINT "fk_99_collab_events_collaborations" FOREIGN KEY ("collaboration_id") REFERENCES "collaborations" ("id");
ALTER TABLE "collab_events" ADD CONSTRAINT "fk_100_collab_events_users" FOREIGN KEY ("actor_user_id") REFERENCES "users" ("id");
ALTER TABLE "work_submissions" ADD CONSTRAINT "fk_101_work_submissions_collaborations" FOREIGN KEY ("collaboration_id") REFERENCES "collaborations" ("id");
ALTER TABLE "disputes" ADD CONSTRAINT "fk_102_disputes_collaborations" FOREIGN KEY ("collaboration_id") REFERENCES "collaborations" ("id");
ALTER TABLE "dispute_decisions" ADD CONSTRAINT "fk_103_dispute_decisions_disputes" FOREIGN KEY ("dispute_id") REFERENCES "disputes" ("id");
ALTER TABLE "dispute_decisions" ADD CONSTRAINT "fk_104_dispute_decisions_users" FOREIGN KEY ("admin_user_id") REFERENCES "users" ("id");
ALTER TABLE "report_actions" ADD CONSTRAINT "fk_105_report_actions_reports" FOREIGN KEY ("report_id") REFERENCES "reports" ("id");
ALTER TABLE "report_actions" ADD CONSTRAINT "fk_106_report_actions_users" FOREIGN KEY ("admin_user_id") REFERENCES "users" ("id");
ALTER TABLE "appeals" ADD CONSTRAINT "fk_107_appeals_users" FOREIGN KEY ("reviewer_user_id") REFERENCES "users" ("id");
ALTER TABLE "reviews" ADD CONSTRAINT "fk_108_reviews_collaborations" FOREIGN KEY ("collaboration_id") REFERENCES "collaborations" ("id");
ALTER TABLE "financial_audit_events" ADD CONSTRAINT "fk_109_financial_audit_events_financial_transactions" FOREIGN KEY ("financial_transaction_id") REFERENCES "financial_transactions" ("id");

COMMIT;
