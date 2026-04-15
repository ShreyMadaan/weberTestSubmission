CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE auctionhouses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) UNIQUE NOT NULL,
    currency CHAR(3) NOT NULL DEFAULT 'USD',
    buyerpremiumpct DECIMAL(5,4) NOT NULL DEFAULT 0.15,
    createdat TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE auctions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    auctionhouseid UUID NOT NULL REFERENCES auctionhouses(id),
    title VARCHAR(255) NOT NULL,
    status VARCHAR(20) NOT NULL,
    scheduledstart TIMESTAMPTZ NOT NULL,
    scheduledend TIMESTAMPTZ NOT NULL,
    actualstart TIMESTAMPTZ,
    actualend TIMESTAMPTZ,
    antisnipewindowsecs SMALLINT NOT NULL DEFAULT 120,
    antisnipeextensionsecs SMALLINT NOT NULL DEFAULT 120,
    allowproxybids BOOLEAN NOT NULL DEFAULT TRUE,
    requireverification BOOLEAN NOT NULL DEFAULT FALSE,
    version INT NOT NULL DEFAULT 1,
    createdat TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE lots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    auctionid UUID NOT NULL REFERENCES auctions(id),
    lotnumber VARCHAR(20) NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(100),
    startingbid BIGINT NOT NULL,
    reserveprice BIGINT,
    currentbid BIGINT NOT NULL,
    currentwinnerid UUID,
    bidcount INT NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL,
    closingat TIMESTAMPTZ,
    soldat TIMESTAMPTZ,
    soldprice BIGINT,
    currency CHAR(3) NOT NULL,
    version INT NOT NULL DEFAULT 1,
    UNIQUE (auctionid, lotnumber)
);

CREATE TABLE bidders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    displayname VARCHAR(100) NOT NULL,
    verificationtier VARCHAR(20) NOT NULL DEFAULT 'basic',
    paddlenumber VARCHAR(20),
    createdat TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE auctionregistrations (
    auctionid UUID NOT NULL REFERENCES auctions(id),
    bidderid UUID NOT NULL REFERENCES bidders(id),
    registeredat TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    depositheld BIGINT,
    iseligible BOOLEAN NOT NULL DEFAULT TRUE,
    PRIMARY KEY (auctionid, bidderid)
);

CREATE TABLE bids (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lotid UUID NOT NULL REFERENCES lots(id),
    bidderid UUID NOT NULL REFERENCES bidders(id),
    bidtype VARCHAR(20) NOT NULL,
    amount BIGINT NOT NULL,
    maxamount BIGINT,
    status VARCHAR(40) NOT NULL,
    idempotencykey VARCHAR(255),
    placedat TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processedat TIMESTAMPTZ,
    outbidat TIMESTAMPTZ,
    CONSTRAINT bid_positive CHECK (amount > 0)
);

CREATE TABLE lotstatetransitions (
    fromstatus VARCHAR(20) NOT NULL,
    tostatus VARCHAR(20) NOT NULL,
    PRIMARY KEY (fromstatus, tostatus)
);

INSERT INTO lotstatetransitions (fromstatus, tostatus) VALUES
    ('pending', 'open'),
    ('open', 'closing'),
    ('open', 'withdrawn'),
    ('closing', 'sold'),
    ('closing', 'passed'),
    ('closing', 'open');

CREATE OR REPLACE FUNCTION enforcelotstatemachine()
RETURNS trigger AS $$
BEGIN
    IF OLD.status = NEW.status THEN
        RETURN NEW;
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM lotstatetransitions
        WHERE fromstatus = OLD.status
          AND tostatus = NEW.status
    ) THEN
        RAISE EXCEPTION 'Invalid lot transition from % to %', OLD.status, NEW.status;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER lotstateguard
BEFORE UPDATE OF status ON lots
FOR EACH ROW
EXECUTE FUNCTION enforcelotstatemachine();

CREATE TABLE bididempotencykeys (
    key VARCHAR(255) NOT NULL,
    bidderid UUID NOT NULL REFERENCES bidders(id),
    responsecode INT NOT NULL DEFAULT 0,
    responsebody JSONB NOT NULL DEFAULT '{}'::jsonb,
    locked BOOLEAN NOT NULL DEFAULT FALSE,
    createdat TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expiresat TIMESTAMPTZ NOT NULL DEFAULT (NOW() + INTERVAL '1 hour'),
    PRIMARY KEY (key, bidderid)
);

CREATE TABLE auctionevents (
    id BIGSERIAL PRIMARY KEY,
    lotid UUID NOT NULL REFERENCES lots(id),
    eventtype VARCHAR(50) NOT NULL,
    payload JSONB NOT NULL,
    occurredat TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE settlementinvoices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    auctionid UUID NOT NULL REFERENCES auctions(id),
    bidderid UUID NOT NULL REFERENCES bidders(id),
    lotid UUID NOT NULL REFERENCES lots(id),
    hammerprice BIGINT NOT NULL,
    buyerpremium BIGINT NOT NULL,
    totaldue BIGINT NOT NULL,
    currency CHAR(3) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    paymentref VARCHAR(100),
    duedate DATE NOT NULL,
    paidat TIMESTAMPTZ,
    createdat TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (lotid)
);

CREATE INDEX idx_auctions_auctionhouseid ON auctions(auctionhouseid);
CREATE INDEX idx_lots_auctionid ON lots(auctionid);
CREATE INDEX idx_lots_status_closingat ON lots(status, closingat);
CREATE INDEX idx_bids_lotid_placedat ON bids(lotid, placedat DESC);
CREATE INDEX idx_bids_bidderid ON bids(bidderid);
CREATE INDEX idx_bididempotencykeys_expiresat ON bididempotencykeys(expiresat);
CREATE INDEX idx_auctionevents_lotid_occurredat ON auctionevents(lotid, occurredat DESC);
CREATE INDEX idx_settlementinvoices_auctionid ON settlementinvoices(auctionid);
CREATE INDEX idx_settlementinvoices_status ON settlementinvoices(status);
