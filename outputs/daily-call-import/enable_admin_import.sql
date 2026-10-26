BEGIN;

CREATE OR REPLACE FUNCTION public.daily_calls_fill_import_metadata()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    admin_id bigint;
    admin_name text;
BEGIN
    -- Normal application inserts already identify their creator.
    IF NEW.created_by_id IS NOT NULL THEN
        RETURN NEW;
    END IF;

    SELECT u.id, COALESCE(NULLIF(BTRIM(u.name), ''), u.username)
      INTO admin_id, admin_name
      FROM public.users u
     WHERE EXISTS (
         SELECT 1 FROM public.user_roles ur
         JOIN public.roles r ON r.id = ur.role_id
         WHERE ur.user_id = u.id AND r.name = 'admin'
     )
     ORDER BY (u.username = 'admin') DESC, u.id
     LIMIT 1;

    IF admin_id IS NULL THEN
        RAISE EXCEPTION 'No administrator account found. Import canceled.';
    END IF;

    NEW.created_by_id := admin_id;
    NEW.created_by_name := admin_name;
    NEW.registration_updated_by_id := admin_id;
    NEW.registration_updated_by_name := admin_name;
    NEW.created_at := COALESCE(NEW.created_at, CURRENT_TIMESTAMP);
    NEW.registration_updated_at := NEW.created_at;
    NEW.updated_at := NEW.created_at;
    NEW.quality_result := COALESCE(NEW.quality_result, '');
    NEW.reason := COALESCE(NEW.reason, '');

    IF BTRIM(NEW.quality_result) <> '' THEN
        NEW.reviewed_by_id := admin_id;
        NEW.reviewed_by_name := admin_name;
        NEW.reviewed_at := COALESCE(NEW.reviewed_at, NEW.created_at);
    ELSE
        NEW.reviewed_by_id := NULL;
        NEW.reviewed_by_name := '';
        NEW.reviewed_at := NULL;
    END IF;

    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS daily_calls_import_metadata ON public.daily_calls;
CREATE TRIGGER daily_calls_import_metadata
BEFORE INSERT ON public.daily_calls
FOR EACH ROW EXECUTE FUNCTION public.daily_calls_fill_import_metadata();

COMMIT;
