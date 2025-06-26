CREATE TABLE IF NOT EXISTS transfer_event (
    event_id_hash INT NOT NULL,
    event_id BIGINT NOT NULL,

    from_wallet VARCHAR(44) NOT NULL,
    to_wallet VARCHAR(44) NOT NULL,

    token VARCHAR(44) NOT NULL,
    amount DECIMAL(20, 0) NOT NULL,
    decimals SMALLINT NOT NULL,

    tx_hash VARCHAR(88) NOT NULL,
    signer VARCHAR(44) NOT NULL,

    block_time INT NOT NULL,
    create_at INT NOT NULL,

    PRIMARY KEY (event_id_hash, event_id)
)
        WITH (CONSISTENCY = 'strong', MUTABILITY = 'MUTABLE_LATEST');

CREATE INDEX IF NOT EXISTS idx_transfer_from_id_desc
    ON transfer_event(from_wallet, event_id DESC);

CREATE INDEX IF NOT EXISTS idx_transfer_to_id_desc
    ON transfer_event(to_wallet, event_id DESC);


