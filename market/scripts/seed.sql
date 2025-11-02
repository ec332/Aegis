-- Seed data for local development and testing

-- Insert sample markets
INSERT INTO markets (id, title, description, status, resolution_datetime, winning_option_id, created_at, updated_at) VALUES
('550e8400-e29b-41d4-a716-446655440001', 
 'Will SpaceX land astronauts on Mars by 2030?', 
 'This market resolves YES if SpaceX successfully lands human astronauts on the surface of Mars before January 1, 2030 00:00:00 UTC.', 
 'active', 
 '2030-01-01 00:00:00', 
 NULL,
 NOW(), 
 NOW()),

('550e8400-e29b-41d4-a716-446655440002', 
 'Will Bitcoin exceed $100,000 in 2025?', 
 'This market resolves YES if Bitcoin (BTC) reaches or exceeds $100,000 USD at any point during calendar year 2025 on major exchanges.', 
 'active', 
 '2025-12-31 23:59:59', 
 NULL,
 NOW(), 
 NOW()),

('550e8400-e29b-41d4-a716-446655440003', 
 'Will there be a major AI breakthrough in 2025?', 
 'This market resolves YES if a significant AI breakthrough is announced by a major research institution or company.', 
 'draft', 
 '2025-12-31 23:59:59', 
 NULL,
 NOW(), 
 NOW());

-- Insert options for Market 1 (SpaceX Mars landing)
INSERT INTO options (id, market_id, title, created_at) VALUES
('650e8400-e29b-41d4-a716-446655440001', '550e8400-e29b-41d4-a716-446655440001', 'Yes', NOW()),
('650e8400-e29b-41d4-a716-446655440002', '550e8400-e29b-41d4-a716-446655440001', 'No', NOW());

-- Insert options for Market 2 (Bitcoin price)
INSERT INTO options (id, market_id, title, created_at) VALUES
('650e8400-e29b-41d4-a716-446655440003', '550e8400-e29b-41d4-a716-446655440002', 'Yes', NOW()),
('650e8400-e29b-41d4-a716-446655440004', '550e8400-e29b-41d4-a716-446655440002', 'No', NOW());

-- Insert options for Market 3 (AI breakthrough)
INSERT INTO options (id, market_id, title, created_at) VALUES
('650e8400-e29b-41d4-a716-446655440005', '550e8400-e29b-41d4-a716-446655440003', 'Yes', NOW()),
('650e8400-e29b-41d4-a716-446655440006', '550e8400-e29b-41d4-a716-446655440003', 'No', NOW());

-- Insert liquidity pools (one per option)
INSERT INTO liquidity_pool (id, market_id, option_id, pool_value, updated_at) VALUES
-- Market 1 pools
('750e8400-e29b-41d4-a716-446655440001', '550e8400-e29b-41d4-a716-446655440001', '650e8400-e29b-41d4-a716-446655440001', 15000.50, NOW()),
('750e8400-e29b-41d4-a716-446655440002', '550e8400-e29b-41d4-a716-446655440001', '650e8400-e29b-41d4-a716-446655440002', 85000.75, NOW()),

-- Market 2 pools
('750e8400-e29b-41d4-a716-446655440003', '550e8400-e29b-41d4-a716-446655440002', '650e8400-e29b-41d4-a716-446655440003', 67500.25, NOW()),
('750e8400-e29b-41d4-a716-446655440004', '550e8400-e29b-41d4-a716-446655440002', '650e8400-e29b-41d4-a716-446655440004', 82500.50, NOW()),

-- Market 3 pools
('750e8400-e29b-41d4-a716-446655440005', '550e8400-e29b-41d4-a716-446655440003', '650e8400-e29b-41d4-a716-446655440005', 30000.00, NOW()),
('750e8400-e29b-41d4-a716-446655440006', '550e8400-e29b-41d4-a716-446655440003', '650e8400-e29b-41d4-a716-446655440006', 30000.00, NOW());

