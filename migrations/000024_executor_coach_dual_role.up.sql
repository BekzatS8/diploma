-- Executors promoted via admin should keep role=executor when they also have coach_profiles.
UPDATE users u
SET role = 'executor',
    updated_at = NOW()
WHERE u.role = 'coach'
  AND EXISTS (SELECT 1 FROM executor_profiles ep WHERE ep.user_id = u.id);
