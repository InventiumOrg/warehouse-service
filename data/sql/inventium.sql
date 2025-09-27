-- Sample data for local development - Milk, Condensed Milk, and Coffee products
INSERT INTO inventory (name, measure, quantity, category, location, unit) VALUES
-- Fresh Milk Products
('Anchor Full Cream Milk 1L', 'L', 120, 'Fresh Milk', 'Cold Storage A', 'can' ),
('Meadow Fresh Lite Milk 1L', 'L', 95, 'Fresh Milk', 'Cold Storage A', 'box'),
('Pura Classic Full Cream 2L', 'L', 80, 'Fresh Milk', 'Cold Storage B', 'can'),
('Dairy Farmers Low Fat Milk 1L', 'L', 110, 'Fresh Milk', 'Cold Storage A', 'package'),
('Paul''s Smarter White Milk 1L', 'L', 75, 'Fresh Milk', 'Cold Storage B', 'package'),
('A2 Platinum Full Cream 1L', 'L', 60, 'Fresh Milk', 'Cold Storage A', 'box'),

-- Condensed Milk Products
('Nestle Sweetened Condensed Milk 395g', 'g', 200, 'Condensed Milk', 'Warehouse A', 'package'),
('Carnation Sweetened Condensed Milk 397g', 'g', 180, 'Condensed Milk', 'Warehouse A', 'box'),
('Top Score Condensed Milk 390g', 'g', 150, 'Condensed Milk', 'Warehouse B', 'package'),
('Ideal Evaporated Milk 410g', 'g', 130, 'Condensed Milk', 'Warehouse A', 'can'),
('Magnolia Sweetened Condensed Milk 300g', 'g', 90, 'Condensed Milk', 'Warehouse B', 'package'),
('Eagle Brand Condensed Milk 397g', 'g', 170, 'Condensed Milk', 'Warehouse A', 'package'),
-- Condensed Milk Products
('Nestle Sweetened Condensed Milk 395g', 'box', 20, 'Condensed Milk', 'Warehouse A', 'box'),
('Carnation Sweetened Condensed Milk 397g', 'box', 180, 'Condensed Milk', 'Warehouse A', 'can'),
('Top Score Condensed Milk 390g', 'box', 50, 'Condensed Milk', 'Warehouse B', 'package'),
('Ideal Evaporated Milk 410g', 'box', 13, 'Condensed Milk', 'Warehouse A', 'package'),
('Magnolia Sweetened Condensed Milk 300g', 'box', 90, 'Condensed Milk', 'Warehouse B', 'box'),
('Eagle Brand Condensed Milk 397g', 'box', 7, 'Condensed Milk', 'Warehouse A', 'package'),

-- Coffee Products
('Nescafe Classic Instant Coffee 175g', 'g', 250, 'Coffee', 'Warehouse C', 'small bag'),
('Moccona Classic Medium Roast 400g', 'g', 180, 'Coffee', 'Warehouse C', 'big bag'),
('International Roast Coffee 500g', 'g', 200, 'Coffee', 'Warehouse D', 'medium bag'),
('Lavazza Qualita Oro Coffee Beans 1kg', 'kg', 75, 'Coffee', 'Warehouse C', 'medium bag'),
('Vittoria Coffee Beans Espresso 1kg', 'kg', 85, 'Coffee', 'Warehouse D', 'small bag'),
('Folgers Classic Roast Ground 326g', 'g', 120, 'Coffee', 'Warehouse C', 'box'),
('Maxwell House Instant Coffee 150g', 'g', 160, 'Coffee', 'Warehouse D', 'box'),
('Starbucks Pike Place Ground 340g', 'g', 95, 'Coffee', 'Warehouse C', 'package'),
('Jacobs Kronung Instant Coffee 200g', 'g', 140, 'Coffee', 'Warehouse D', 'medium bag'),
('Douwe Egberts Pure Gold 190g', 'g', 110, 'Coffee', 'Warehouse C', 'medium bag'),

-- UHT/Long Life Milk
('Devondale Long Life Full Cream 1L', 'L', 300, 'UHT Milk', 'Warehouse E', 'box'),
('Sanitarium So Good Soy Milk 1L', 'L', 180, 'UHT Milk', 'Warehouse E', 'box'),
('Vitasoy Almond Milk 1L', 'L', 150, 'UHT Milk', 'Warehouse E', 'box'),
('Australia''s Own Oat Milk 1L', 'L', 120, 'UHT Milk', 'Warehouse E', 'can'),

-- Premium Coffee
('Blue Mountain Coffee Beans 250g', 'g', 45, 'Premium Coffee', 'Warehouse F', 'medium bag'),
('Kona Coffee Ground 227g', 'g', 35, 'Premium Coffee', 'Warehouse F', 'big bag'),
('Ethiopian Yirgacheffe Beans 500g', 'g', 60, 'Premium Coffee', 'Warehouse F', 'small bag');