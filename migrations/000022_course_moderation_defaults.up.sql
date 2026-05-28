UPDATE courses
SET moderation_status = 'approved'
WHERE status = 'published'
  AND moderation_status <> 'approved';

UPDATE courses
SET moderation_status = 'draft'
WHERE status = 'draft'
  AND moderation_status = 'approved';
