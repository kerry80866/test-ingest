CREATE TABLE IF NOT EXISTS balance (
    account_address VARCHAR(44) NOT NULL,
    owner_address   VARCHAR(44) NOT NULL,
    token_address   VARCHAR(44) NOT NULL,
    balance         DECIMAL(20, 0) NOT NULL,
    last_event_id   BIGINT NOT NULL
    PRIMARY KEY (account_address)
) WITH (CONSISTENCY = 'strong', MUTABILITY = 'MUTABLE_LATEST');

CREATE INDEX IF NOT EXISTS idx_balance_token_owner
    ON balance(token_address, owner_address)
    INCLUDE (balance)

CREATE INDEX IF NOT EXISTS idx_balance_token_balance
    ON balance(token_address, balance DESC)
    INCLUDE (owner_address)

CREATE INDEX IF NOT EXISTS idx_balance_owner_token
    ON balance(owner_address, token_address)
    INCLUDE (balance);
