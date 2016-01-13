-- +migrate Up
UPDATE client_identity
SET metadata = text(
    json_build_object(
        'redirectURLs', json(json(metadata)->>'redirectURLs'),
        'redirect_uris', json(json(metadata)->>'redirectURLs')
    )
 )
WHERE (json(metadata)->>'redirect_uris') IS NULL;
