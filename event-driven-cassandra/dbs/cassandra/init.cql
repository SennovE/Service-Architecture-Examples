CREATE KEYSPACE IF NOT EXISTS warehouse
WITH REPLICATION = {
  'class': 'NetworkTopologyStrategy',
  'datacenter1': 3
}
AND DURABLE_WRITES = true;

USE warehouse;

CREATE TABLE IF NOT EXISTS inventory_by_product_zone (
    product_id TEXT,
    zone_id TEXT,
    available_quantity INT,
    reserved_quantity INT,
    supplier_id TEXT,
    PRIMARY KEY ((product_id), zone_id)
);

CREATE TABLE IF NOT EXISTS inventory_by_zone (
    zone_id TEXT,
    product_id TEXT,
    available_quantity INT,
    reserved_quantity INT,
    supplier_id TEXT,
    PRIMARY KEY ((zone_id), product_id)
);

CREATE TABLE IF NOT EXISTS inventory_totals_by_product (
    product_id TEXT,
    total_available_quantity INT,
    total_reserved_quantity INT,
    supplier_id TEXT,
    PRIMARY KEY ((product_id))
);

CREATE TABLE IF NOT EXISTS processed_events (
    event_id TEXT,
    PRIMARY KEY ((event_id))
);

CREATE TABLE IF NOT EXISTS last_seq_number (
    enity_id TEXT,
    sequence_number BIGINT,
    PRIMARY KEY ((enity_id))
);

CREATE TABLE IF NOT EXISTS orders_by_id (
    order_id TEXT,
    status TEXT,
    PRIMARY KEY ((order_id))
);

CREATE TABLE IF NOT EXISTS order_items_by_order (
    order_id TEXT,
    product_id TEXT,
    zone_id TEXT,
    quantity INT,
    PRIMARY KEY ((order_id), product_id, zone_id)
);
