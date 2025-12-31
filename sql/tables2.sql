-- Run these on different servers to create GTID differences

-- Scenario 1: Create some transactions on source only
-- (Run this only on your source server)
CALL CreateBulkTestData(5, 10);

-- Scenario 2: Create conflicting updates
-- (Run these simultaneously on different servers)
CALL SimulateConflict(1, 100.00);
CALL SimulateConflict(2, -50.00);

-- Scenario 3: Create some DDL changes
-- (Run on one server only to create schema differences)
ALTER TABLE users ADD COLUMN last_login TIMESTAMP NULL;
ALTER TABLE products ADD COLUMN weight DECIMAL(5,2) DEFAULT 0.00;

-- Scenario 4: Bulk operations for GTID testing
INSERT INTO chaos2.audit_log (table_name, record_id, action, new_values) 
SELECT 'bulk_test', id, 'INSERT', JSON_OBJECT('test', 'bulk_operation') 
FROM chaos2.users LIMIT 10;