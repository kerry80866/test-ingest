CREATE TABLE IF NOT EXISTS token (
    token_address VARCHAR(44) NOT NULL,
    decimals SMALLINT NOT NULL,
    source SMALLINT NOT NULL,
    total_supply DECIMAL(20, 0) NOT NULL,
    name VARCHAR(128) NOT NULL,
    symbol VARCHAR(64) NOT NULL,
    creator VARCHAR(44) NOT NULL,
    uri VARCHAR(256) NOT NULL,

    create_at INT NOT NULL,
    update_at INT NOT NULL,
    PRIMARY KEY (token_address)
) WITH (CONSISTENCY = 'strong', MUTABILITY = 'MUTABLE_LATEST');

