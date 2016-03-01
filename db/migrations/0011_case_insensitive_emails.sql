-- +migrate Up

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
