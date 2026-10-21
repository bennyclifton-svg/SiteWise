-- Source readings are separate from accepted comparison/contract prices.
ALTER TABLE submissions ADD COLUMN quote_review jsonb;
