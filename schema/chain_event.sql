CREATE TABLE IF NOT EXISTS chain_event (
     event_id_hash INT NOT NULL,
     event_id BIGINT NOT NULL,
     event_type SMALLINT NOT NULL,
     dex SMALLINT NOT NULL ,

     user_wallet VARCHAR(44) NOT NULL,
     to_wallet VARCHAR(44) NOT NULL,

     pool_address VARCHAR(44) NOT NULL,
     token VARCHAR(44) NOT NULL,
     quote_token VARCHAR(44) NOT NULL,

     token_amount DECIMAL(20, 0) NOT NULL,
     quote_amount DECIMAL(20, 0) NOT NULL,
     volume_usd DOUBLE NOT NULL,
     price_usd DOUBLE NOT NULL,

     tx_hash VARCHAR(88) NOT NULL,
     signer VARCHAR(44) NOT NULL,

     block_time INT NOT NULL,
     create_at INT NOT NULL,

     PRIMARY KEY (event_id_hash, event_id)
)
        WITH (CONSISTENCY = 'strong', MUTABILITY = 'MUTABLE_LATEST');

CREATE INDEX IF NOT EXISTS idx_user_token_type_id_desc
    ON chain_event(user_wallet, token, event_type, event_id DESC)
    WITH (INDEX_COVERED_TYPE = 'COVERED_ALL_COLUMNS_IN_SCHEMA');

CREATE INDEX IF NOT EXISTS idx_pool_type_id
    ON chain_event(pool_address, event_type, event_id DESC)
    WITH (INDEX_COVERED_TYPE = 'COVERED_ALL_COLUMNS_IN_SCHEMA');

CREATE INDEX IF NOT EXISTS idx_user_type_time
    ON chain_event(user_wallet, event_type, block_time DESC);

--     WITH (INDEX_COVERED_TYPE = 'COVERED_ALL_COLUMNS_IN_SCHEMA');
