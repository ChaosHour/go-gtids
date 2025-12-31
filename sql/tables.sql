-- Create the chaos2 schema
CREATE SCHEMA IF NOT EXISTS chaos2;
USE chaos2;

-- Table 1: Users table with various data types
CREATE TABLE users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    email VARCHAR(100) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    is_active BOOLEAN DEFAULT TRUE,
    age INT,
    balance DECIMAL(10,2) DEFAULT 0.00,
    INDEX idx_username (username),
    INDEX idx_email (email),
    INDEX idx_created_at (created_at)
);

-- Table 2: Orders table for testing foreign keys and transactions
CREATE TABLE orders (
    id INT AUTO_INCREMENT PRIMARY KEY,
    user_id INT NOT NULL,
    order_number VARCHAR(20) NOT NULL UNIQUE,
    total_amount DECIMAL(10,2) NOT NULL,
    status ENUM('pending', 'processing', 'shipped', 'delivered', 'cancelled') DEFAULT 'pending',
    order_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    ship_date TIMESTAMP NULL,
    notes TEXT,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_user_id (user_id),
    INDEX idx_order_number (order_number),
    INDEX idx_status (status),
    INDEX idx_order_date (order_date)
);

-- Table 3: Products table
CREATE TABLE products (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    price DECIMAL(8,2) NOT NULL,
    stock_quantity INT DEFAULT 0,
    category VARCHAR(50),
    sku VARCHAR(50) UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_name (name),
    INDEX idx_category (category),
    INDEX idx_sku (sku)
);

-- Table 4: Order items (junction table)
CREATE TABLE order_items (
    id INT AUTO_INCREMENT PRIMARY KEY,
    order_id INT NOT NULL,
    product_id INT NOT NULL,
    quantity INT NOT NULL DEFAULT 1,
    unit_price DECIMAL(8,2) NOT NULL,
    total_price DECIMAL(10,2) GENERATED ALWAYS AS (quantity * unit_price) STORED,
    FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE,
    FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE,
    INDEX idx_order_id (order_id),
    INDEX idx_product_id (product_id)
);

-- Table 5: Audit log for testing triggers and bulk operations
CREATE TABLE audit_log (
    id INT AUTO_INCREMENT PRIMARY KEY,
    table_name VARCHAR(50) NOT NULL,
    record_id INT NOT NULL,
    action ENUM('INSERT', 'UPDATE', 'DELETE') NOT NULL,
    old_values JSON,
    new_values JSON,
    user_id INT,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_table_record (table_name, record_id),
    INDEX idx_action (action),
    INDEX idx_timestamp (timestamp)
);

-- Insert test data for users
INSERT INTO users (username, email, password_hash, age, balance, is_active) VALUES
('john_doe', 'john@example.com', '$2b$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/Lrxyr8dDnqABCDEFG', 28, 150.75, TRUE),
('jane_smith', 'jane@example.com', '$2b$12$XQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/Lrxyr8dDnqABCDEFH', 34, 89.50, TRUE),
('bob_wilson', 'bob@example.com', '$2b$12$YQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/Lrxyr8dDnqABCDEFI', 45, 0.00, FALSE),
('alice_brown', 'alice@example.com', '$2b$12$ZQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/Lrxyr8dDnqABCDEFJ', 29, 225.00, TRUE),
('charlie_davis', 'charlie@example.com', '$2b$12$AQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/Lrxyr8dDnqABCDEFK', 52, 1750.25, TRUE),
('diana_miller', 'diana@example.com', '$2b$12$BQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/Lrxyr8dDnqABCDEFL', 31, 67.80, TRUE),
('frank_garcia', 'frank@example.com', '$2b$12$CQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/Lrxyr8dDnqABCDEFM', 38, 0.00, TRUE),
('grace_martinez', 'grace@example.com', '$2b$12$DQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/Lrxyr8dDnqABCDEFN', 26, 892.15, TRUE);

-- Insert test data for products
INSERT INTO products (name, description, price, stock_quantity, category, sku) VALUES
('Laptop Pro 15"', 'High-performance laptop with 16GB RAM', 1299.99, 25, 'Electronics', 'LAP-PRO-15-001'),
('Wireless Mouse', 'Ergonomic wireless mouse with USB receiver', 29.99, 150, 'Electronics', 'MSE-WRL-001'),
('Office Chair', 'Ergonomic office chair with lumbar support', 249.50, 45, 'Furniture', 'CHR-OFF-001'),
('Coffee Mug', 'Ceramic coffee mug 12oz', 12.95, 200, 'Kitchen', 'MUG-CER-12-001'),
('Smartphone X', 'Latest smartphone with 128GB storage', 699.00, 75, 'Electronics', 'PHN-SMX-128-001'),
('Desk Lamp', 'LED desk lamp with adjustable brightness', 45.00, 60, 'Furniture', 'LMP-DSK-LED-001'),
('Water Bottle', 'Stainless steel water bottle 24oz', 24.95, 120, 'Sports', 'BTL-STL-24-001'),
('Notebook', 'Spiral notebook 200 pages', 8.50, 300, 'Office', 'NTB-SPR-200-001'),
('Headphones', 'Noise-cancelling over-ear headphones', 199.99, 40, 'Electronics', 'HPH-NOI-001'),
('Backpack', 'Laptop backpack with multiple compartments', 79.99, 85, 'Travel', 'BPK-LAP-001');

-- Insert test data for orders
INSERT INTO orders (user_id, order_number, total_amount, status, ship_date, notes) VALUES
(1, 'ORD-2024-001', 1329.98, 'delivered', '2024-06-01 10:30:00', 'Express shipping requested'),
(2, 'ORD-2024-002', 57.90, 'shipped', '2024-06-05 14:15:00', NULL),
(4, 'ORD-2024-003', 294.50, 'processing', NULL, 'Customer requested blue color'),
(1, 'ORD-2024-004', 729.00, 'pending', NULL, 'Bulk order discount applied'),
(5, 'ORD-2024-005', 199.99, 'delivered', '2024-05-28 09:45:00', 'Gift wrap requested'),
(6, 'ORD-2024-006', 104.90, 'cancelled', NULL, 'Customer changed mind'),
(4, 'ORD-2024-007', 12.95, 'delivered', '2024-06-03 16:20:00', NULL),
(8, 'ORD-2024-008', 1575.48, 'processing', NULL, 'Large order - special handling');

-- Insert test data for order_items
INSERT INTO order_items (order_id, product_id, quantity, unit_price) VALUES
-- Order 1: Laptop + Mouse
(1, 1, 1, 1299.99),
(1, 2, 1, 29.99),
-- Order 2: Office supplies
(2, 4, 2, 12.95),
(2, 8, 4, 8.50),
-- Order 3: Office Chair + Desk Lamp
(3, 3, 1, 249.50),
(3, 6, 1, 45.00),
-- Order 4: Smartphone
(4, 5, 1, 699.00),
(4, 2, 1, 29.99),
-- Order 5: Headphones
(5, 9, 1, 199.99),
-- Order 6: Water Bottles (cancelled order)
(6, 7, 3, 24.95),
(6, 8, 5, 8.50),
-- Order 7: Coffee Mug
(7, 4, 1, 12.95),
-- Order 8: Multiple items
(8, 1, 1, 1299.99),
(8, 9, 1, 199.99),
(8, 10, 1, 79.99);

-- Insert some audit log entries
INSERT INTO audit_log (table_name, record_id, action, new_values, user_id) VALUES
('users', 1, 'INSERT', '{"username": "john_doe", "email": "john@example.com"}', NULL),
('users', 2, 'INSERT', '{"username": "jane_smith", "email": "jane@example.com"}', NULL),
('orders', 1, 'INSERT', '{"order_number": "ORD-2024-001", "total_amount": 1329.98}', 1),
('orders', 2, 'UPDATE', '{"status": "shipped", "ship_date": "2024-06-05 14:15:00"}', 2);

-- Create some useful procedures for testing GTID scenarios
DELIMITER //

-- Procedure to create bulk test data (useful for GTID testing)
CREATE PROCEDURE CreateBulkTestData(IN num_users INT, IN num_orders INT)
BEGIN
    DECLARE i INT DEFAULT 0;
    DECLARE user_count INT;
    
    SELECT COUNT(*) INTO user_count FROM users;
    
    -- Create additional users
    WHILE i < num_users DO
        INSERT INTO users (username, email, password_hash, age, balance) 
        VALUES (
            CONCAT('testuser_', user_count + i + 1),
            CONCAT('test', user_count + i + 1, '@example.com'),
            '$2b$12$TestHashForBulkData123456789',
            FLOOR(18 + RAND() * 50),
            ROUND(RAND() * 1000, 2)
        );
        SET i = i + 1;
    END WHILE;
    
    -- Create additional orders
    SET i = 0;
    WHILE i < num_orders DO
        INSERT INTO orders (user_id, order_number, total_amount, status)
        VALUES (
            1 + FLOOR(RAND() * (user_count + num_users)),
            CONCAT('BULK-', YEAR(NOW()), '-', LPAD(i + 1, 6, '0')),
            ROUND(10 + RAND() * 500, 2),
            ELT(1 + FLOOR(RAND() * 5), 'pending', 'processing', 'shipped', 'delivered', 'cancelled')
        );
        SET i = i + 1;
    END WHILE;
END//

-- Procedure to simulate transaction conflicts (useful for GTID errant transaction testing)
CREATE PROCEDURE SimulateConflict(IN user_id INT, IN amount DECIMAL(10,2))
BEGIN
    DECLARE EXIT HANDLER FOR SQLEXCEPTION
    BEGIN
        ROLLBACK;
        RESIGNAL;
    END;
    
    START TRANSACTION;
    
    -- Update user balance
    UPDATE users SET balance = balance + amount WHERE id = user_id;
    
    -- Log the transaction
    INSERT INTO audit_log (table_name, record_id, action, new_values, user_id)
    VALUES ('users', user_id, 'UPDATE', JSON_OBJECT('balance_change', amount), user_id);
    
    -- Simulate some processing time
    SELECT SLEEP(0.1);
    
    COMMIT;
END//

DELIMITER ;

-- Create a view for testing
CREATE VIEW user_order_summary AS
SELECT 
    u.id,
    u.username,
    u.email,
    u.balance,
    COUNT(o.id) as total_orders,
    COALESCE(SUM(o.total_amount), 0) as total_spent,
    MAX(o.order_date) as last_order_date
FROM users u
LEFT JOIN orders o ON u.id = o.user_id
GROUP BY u.id, u.username, u.email, u.balance;

-- Show summary of created data
SELECT 'Data Creation Summary' as Info;
SELECT COUNT(*) as user_count FROM users;
SELECT COUNT(*) as product_count FROM products;
SELECT COUNT(*) as order_count FROM orders;
SELECT COUNT(*) as order_item_count FROM order_items;
SELECT COUNT(*) as audit_log_count FROM audit_log;