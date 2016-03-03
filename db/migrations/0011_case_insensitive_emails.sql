-- +migrate Up

-- This migration is a fix for a bug that allowed duplicate emails if they used different cases (see #338).
-- When migrating, dex will not take the liberty of deleting rows for duplicate cases. Instead it will
-- raise an exception and call for an admin to remove duplicates manually.

CREATE OR REPLACE FUNCTION raise_exp() RETURNS VOID AS $$
BEGIN
     RAISE EXCEPTION 'Found duplicate emails when using case insensitive comparision, cannot perform migration.';
END;
$$ LANGUAGE plpgsql;

SELECT LOWER(email),
    COUNT(email),
    CASE
        WHEN COUNT(email) > 1 THEN raise_exp()
        ELSE NULL
    END
FROM authd_user
GROUP BY LOWER(email);

UPDATE authd_user SET email = LOWER(email);
