-- Проект дополнений к DBML; не миграция. Применяется после создания таблиц.
ALTER TABLE companies ADD CONSTRAINT ck_company_version CHECK (version > 0),
  ADD CONSTRAINT ck_company_name CHECK (length(btrim(name)) > 0),
  ADD CONSTRAINT ck_company_block CHECK (lifecycle <> 'BLOCKED' OR (blocked_at IS NOT NULL AND blocked_by IS NOT NULL AND block_reason IS NOT NULL AND length(btrim(block_reason)) > 0));
ALTER TABLE company_members ADD CONSTRAINT ck_member_version CHECK (version > 0);
ALTER TABLE company_invitations ADD CONSTRAINT ck_invitation_version CHECK (version > 0),
  ADD CONSTRAINT ck_invitation_expiry CHECK (expires_at > created_at),
  ADD CONSTRAINT ck_invitation_accepted CHECK (status <> 'ACCEPTED' OR accepted_at IS NOT NULL),
  ADD CONSTRAINT ck_invitation_role CHECK (proposed_role <> 'OWNER');
CREATE UNIQUE INDEX ux_invitation_pending ON company_invitations (company_id, invitee_identity_id) WHERE status = 'PENDING';
ALTER TABLE verification_requests ADD CONSTRAINT ck_verification_version CHECK (version > 0 AND ordinal > 0),
  ADD CONSTRAINT ck_verification_decision CHECK (status NOT IN ('APPROVED', 'REJECTED', 'SUPERSEDED') OR (decided_by IS NOT NULL AND decided_at IS NOT NULL));
CREATE UNIQUE INDEX ux_verification_pending ON verification_requests (company_id) WHERE status = 'PENDING';
ALTER TABLE companies ADD CONSTRAINT ck_company_request_pair_1 CHECK ((create_request_id IS NULL) = (create_request_hash IS NULL));
ALTER TABLE company_invitations ADD CONSTRAINT ck_company_request_pair_2 CHECK ((create_request_id IS NULL) = (create_request_hash IS NULL));
ALTER TABLE verification_requests ADD CONSTRAINT ck_company_request_pair_3 CHECK ((create_request_id IS NULL) = (create_request_hash IS NULL));
