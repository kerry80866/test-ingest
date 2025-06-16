CREATE TABLE IF NOT EXISTS pool (
    pool_address VARCHAR(44) NOT NULL,
    dex SMALLINT NOT NULL,
    token_address VARCHAR(44) NOT NULL,
    quote_address VARCHAR(44) NOT NULL,
    token_account VARCHAR(44) NOT NULL,
    quote_account VARCHAR(44) NOT NULL,
    create_at INT NOT NULL,
    update_at INT NOT NULL,
    PRIMARY KEY (pool_address, token_account, quote_account)
) WITH (CONSISTENCY = 'strong', MUTABILITY = 'MUTABLE_LATEST');

CREATE INDEX IF NOT EXISTS idx_pool_token_quote
    ON pool(token_address, quote_address)
    WITH (INDEX_COVERED_TYPE = 'COVERED_ALL_COLUMNS_IN_SCHEMA');
