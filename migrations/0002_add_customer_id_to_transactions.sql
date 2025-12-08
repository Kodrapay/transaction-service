-- Add customer_id column to the transactions table
ALTER TABLE transactions
ADD COLUMN customer_id BIGINT;