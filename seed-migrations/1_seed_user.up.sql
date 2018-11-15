INSERT INTO users (
        username,
        hashed_password,
        first_name,
        last_name,
        roles
    )
    VALUES (
        'test',
        '$2a$04$oLK1h.TmOdsz6PUszjzj3eEjCXdoz8RIh5Q8lIb5aTmVtLCNK.DTG',
        'John',
        'Doe',
        '{"isVerified": true, "isAdmin": true}'
    )