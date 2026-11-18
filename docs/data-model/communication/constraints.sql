-- Проект дополнений к DBML, не миграция.
ALTER TABLE conversations ADD CONSTRAINT ck_conversation_version CHECK (version > 0 AND next_sequence > 0),
 ADD CONSTRAINT ck_conversation_type CHECK (type IN ('DIRECT','APPLICATION','INTERVIEW')),
 ADD CONSTRAINT ck_conversation_context CHECK ((type = 'DIRECT' AND context_id IS NULL) OR (type IN ('APPLICATION','INTERVIEW') AND context_id IS NOT NULL)),
 ADD CONSTRAINT ck_conversation_status CHECK (status IN ('ACTIVE','READ_ONLY','ARCHIVED'));
ALTER TABLE direct_conversation_pairs ADD CONSTRAINT ck_direct_order CHECK (identity_low < identity_high);
ALTER TABLE conversation_participants ADD CONSTRAINT ck_participant_status CHECK (status IN ('ACTIVE','REMOVED')),
 ADD CONSTRAINT ck_participant_removed CHECK (status <> 'REMOVED' OR removed_at IS NOT NULL);
ALTER TABLE messages ADD CONSTRAINT ck_message_version CHECK (version > 0 AND sequence > 0),
 ADD CONSTRAINT ck_message_status CHECK (status IN ('SENT','EDITED','SOFT_DELETED')),
 ADD CONSTRAINT ck_message_deleted CHECK (status <> 'SOFT_DELETED' OR (content IS NULL AND deleted_at IS NOT NULL)),
 ADD CONSTRAINT ck_message_reply CHECK (reply_to_message_id IS NULL OR reply_to_message_id <> id);
ALTER TABLE message_attachments ADD CONSTRAINT ck_attachment_size CHECK (size_bytes > 0);
ALTER TABLE message_reactions ADD CONSTRAINT ck_emoji_not_blank CHECK (length(btrim(emoji_code)) > 0);
ALTER TABLE attachment_cleanup_jobs ADD CONSTRAINT ck_cleanup_attempts CHECK (attempts >= 0), ADD CONSTRAINT ck_cleanup_lease CHECK ((lease_token IS NULL) = (lease_until IS NULL));
ALTER TABLE conversations ADD CONSTRAINT ck_communication_request_pair_1 CHECK ((create_request_id IS NULL) = (create_request_hash IS NULL));
ALTER TABLE messages ADD CONSTRAINT ck_communication_request_pair_2 CHECK ((create_request_id IS NULL) = (create_request_hash IS NULL));
