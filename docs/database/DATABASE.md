# Zedlancer — Database Technical Documentation

**Database:** PostgreSQL  
**Source of truth:** `schema.dbml` (supplied merged v4 base / v5-enhanced model)  
**Inventory:** 68 tables · 109 explicit relationships · 21 PostgreSQL enum types

## 1. Repository Files

- `schema.dbml` — editable full schema / ERD source, including column types and relationship definitions.
- `erd.svg` — rendered ERD with all tables, columns, types, and explicit relationships.
- `schema.sql` — PostgreSQL DDL translated from the DBML (tables, types, primary keys, unique constraints, foreign keys, and column defaults).
- `DATABASE.md` — this implementation and normalization guide.

## 2. Schema Inventory

Every table is included in `schema.dbml` and in `schema.sql`. The ERD connects foreign keys exactly as defined in the source.

**Identity and Profiles (12):** `users`, `addresses`, `parents`, `job_seekers`, `employers`, `individual_employers`, `legal_employers`, `user_sessions`, `admin_profiles`, `parent_links`, `parent_invitations`, `parent_link_requests`.

**Skills, Growth and Files (8):** `skills`, `job_seeker_skills`, `skill_evidence`, `portfolio_items`, `challenges`, `challenge_attempts`, `growth_goals`, `files`.

**Opportunities and Applications (13):** `job_postings`, `job_posting_versions`, `job_posting_skills`, `job_posting_version_skills`, `job_applications`, `offers`, `internship_details`, `screening_results`, `saved_job_postings`, `categories`, `activities`, `activity_categories`, `availability_slots`.

**Contracts and Collaboration (15):** `work_contracts`, `contract_versions`, `contract_approvals`, `contract_milestones`, `contract_version_acceptances`, `contract_parent_consents`, `collaborations`, `tasks`, `deliverables`, `deliverable_files`, `revision_requests`, `stop_requests`, `extension_requests`, `collab_events`, `work_submissions`.

**Messages and Moderation (14):** `conversations`, `messages`, `message_attachments`, `collaboration_messages`, `collaboration_message_files`, `verification_documents`, `admin_actions`, `disputes`, `dispute_evidence`, `dispute_decisions`, `reports`, `report_actions`, `appeals`, `reviews`.

**Payments and Audit (4):** `payments`, `payouts`, `financial_transactions`, `financial_audit_events`.

**Other Tables (2):** `notifications`, `action_items`.

## 3. Keys, Data Types and Relationships

- Primary keys: represented by `[pk]` or composite `indexes { (...) [pk] }` in DBML and by `PRIMARY KEY` in SQL.
- Foreign keys: all explicit `Ref` definitions are mapped to `FOREIGN KEY` constraints. They are added after table creation to support circular links.
- Unique keys: column-level `[unique]` and composite unique indexes are preserved.
- Types: UUID, integer, numeric, boolean, date/timestamp, text/varchar, JSONB, and named PostgreSQL enums follow the source model.
- Nullability and defaults: source `not null` and `default` annotations are preserved; unspecified values remain nullable unless covered by a primary key.

## 4. Normalization and Deliberate Exceptions

**First normal form (1NF).** The schema uses individually keyed tables and bridge tables for many-to-many associations (for example, `job_seeker_skills`, `job_posting_skills` and `deliverable_files`). Some `jsonb` fields store structured content that cannot be analyzed for atomicity from DBML alone.

**Second normal form (2NF).** Single-column-keyed tables have no partial dependency on a composite primary key. Composite-key bridge tables represent relationships; their non-key dependencies must still be validated against business rules.

**Third normal form (3NF).** Identity/authentication is separated from profile entities; employer subtypes and reusable catalogs are modeled separately; contract and job-posting versions are maintained in dedicated tables. **The DBML alone cannot prove full 3NF:** this requires functional dependencies and candidate-key review against actual business rules.

**Intentional denormalization / flexible data.** `job_postings.current_version_id`, `work_contracts.current_version_id`, and `collaborations.active_contract_version_id` enable direct current-state lookup while historical versions remain in version tables. `jsonb` (`contract_versions.terms` and other payload/annotation fields) supports variable shape data. These fields require consistency checks and schema validation in application logic or additional migrations.

## 5. Integrity and Implementation Limits

The following requirements are documented in source notes but **are not fully defined as executable constraints in the supplied DBML**:

- Version pointer must reference a version owned by the same posting/contract; implement composite keys/FKs or triggers.
- Active parent–teen linkage and appropriate consent are required when applicable; version changes may require fresh consent.
- Polymorphic actor/recipient/owner identifiers require type-appropriate validation.
- Age, monetary ranges, time intervals, state transitions, conditional uniqueness and partial indexes require explicit PostgreSQL migrations.
- Audit/evidence append-only behavior, deletion restrictions, and idempotency policies need application and/or database enforcement.
- Real monetary transactions remain in existing `payments`, `payouts` and `financial_transactions`; event additions do not replace them.

**Important:** `schema.sql` deliberately does not invent missing CHECK constraints, partial indexes or triggers. It is a structural translation, not a guarantee of all business rules.

## 6. Deploy / Reproduce

For an empty PostgreSQL database:

```bash
psql -v ON_ERROR_STOP=1 -U <username> -d <database> -f database/schema.sql
```

Apply any additional project-specific migrations for the integrity rules above. DBML and the SQL script should be kept in sync when the schema changes.
