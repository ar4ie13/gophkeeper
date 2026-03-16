BEGIN TRANSACTION;

CREATE TABLE users (
                       uuid UUID PRIMARY KEY DEFAULT (gen_random_uuid()),
                       login VARCHAR UNIQUE NOT NULL,
                       password_hash VARCHAR NOT NULL,
                       encrypted_key BYTEA NOT NULL,
                       created_at TIMESTAMP DEFAULT (CURRENT_TIMESTAMP),
                       updated_at TIMESTAMP DEFAULT (CURRENT_TIMESTAMP)
);

CREATE TABLE text (
                       name VARCHAR NOT NULL,
                       description VARCHAR NOT NULL,
                       user_uuid UUID NOT NULL,
                       user_text BYTEA NOT NULL,
                       created_at TIMESTAMP DEFAULT (CURRENT_TIMESTAMP),
                       updated_at TIMESTAMP DEFAULT (CURRENT_TIMESTAMP),
                       version INTEGER NOT NULL DEFAULT 1,
                       PRIMARY KEY (user_uuid, name),
                       CONSTRAINT fk_users
                           FOREIGN KEY (user_uuid)
                               REFERENCES users (uuid)
);

CREATE TABLE credential (
                      name VARCHAR NOT NULL,
                      description VARCHAR NOT NULL,
                      user_uuid UUID NOT NULL,
                      login BYTEA NOT NULL,
                      password BYTEA NOT NULL,
                      created_at TIMESTAMP DEFAULT (CURRENT_TIMESTAMP),
                      updated_at TIMESTAMP DEFAULT (CURRENT_TIMESTAMP),
                      version INTEGER NOT NULL DEFAULT 1,
                      PRIMARY KEY (user_uuid, name),
                      CONSTRAINT fk_users
                          FOREIGN KEY (user_uuid)
                              REFERENCES users (uuid)
);

CREATE TABLE card (
                            name VARCHAR NOT NULL,
                            description VARCHAR NOT NULL,
                            user_uuid UUID NOT NULL,
                            card_number BYTEA NOT NULL,
                            owner BYTEA NOT NULL,
                            expires_at BYTEA NOT NULL,
                            cvc BYTEA NOT NULL,
                            created_at TIMESTAMP DEFAULT (CURRENT_TIMESTAMP),
                            updated_at TIMESTAMP DEFAULT (CURRENT_TIMESTAMP),
                            version INTEGER NOT NULL DEFAULT 1,
                            PRIMARY KEY (user_uuid, name),
                            CONSTRAINT fk_users
                                FOREIGN KEY (user_uuid)
                                    REFERENCES users (uuid)

);

CREATE TABLE file (
                      name VARCHAR NOT NULL,
                      description VARCHAR NOT NULL,
                      user_uuid UUID NOT NULL,
                      user_file BYTEA NOT NULL,
                      created_at TIMESTAMP DEFAULT (CURRENT_TIMESTAMP),
                      updated_at TIMESTAMP DEFAULT (CURRENT_TIMESTAMP),
                      version INTEGER NOT NULL DEFAULT 1,
                      PRIMARY KEY (user_uuid, name),
                      CONSTRAINT fk_users
                          FOREIGN KEY (user_uuid)
                              REFERENCES users (uuid)

);

    COMMIT;