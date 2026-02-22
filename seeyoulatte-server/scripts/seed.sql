-- Seed script for SeeYouLatte
-- Creates test users and 50 coffee listings

-- Clear existing data (in reverse order of foreign key dependencies)
TRUNCATE TABLE reviews CASCADE;
TRUNCATE TABLE disputes CASCADE;
TRUNCATE TABLE ledger_entries CASCADE;
TRUNCATE TABLE orders CASCADE;
TRUNCATE TABLE listings CASCADE;
TRUNCATE TABLE users CASCADE;

-- Create test users (sellers and buyers)
-- All users have password: "password123"
INSERT INTO users (id, email, password_hash, name, bio, location_text, is_verified, created_at) VALUES
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'sarah.coffeelover@example.com', '$2a$10$2WiBXZWqZXLIk3RKOtvRz.BPe.OB.ACv6JQWtofpUowLVMGihAjwy', 'Sarah Chen', 'Home roaster since 2019, specializing in light roasts. I love experimenting with different origins and processing methods.', 'Xinyi District, Taipei', true, NOW() - INTERVAL '6 months'),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12', 'mike.espresso@example.com', '$2a$10$2WiBXZWqZXLIk3RKOtvRz.BPe.OB.ACv6JQWtofpUowLVMGihAjwy', 'Mike Johnson', 'La Marzocco owner, pulling shots since 2015. Come try my signature espresso blends!', 'Da''an District, Taipei', true, NOW() - INTERVAL '8 months'),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13', 'emma.brewmaster@example.com', '$2a$10$2WiBXZWqZXLIk3RKOtvRz.BPe.OB.ACv6JQWtofpUowLVMGihAjwy', 'Emma Liu', 'V60 specialist, SCAA certified. I source beans directly from farmers in Yunnan.', 'Zhongshan District, Taipei', true, NOW() - INTERVAL '4 months'),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a14', 'david.roaster@example.com', '$2a$10$2WiBXZWqZXLIk3RKOtvRz.BPe.OB.ACv6JQWtofpUowLVMGihAjwy', 'David Wong', 'Professional roaster with 10+ years experience. Small batch, carefully crafted.', 'Songshan District, Taipei', true, NOW() - INTERVAL '1 year'),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a15', 'lisa.coffee@example.com', '$2a$10$2WiBXZWqZXLIk3RKOtvRz.BPe.OB.ACv6JQWtofpUowLVMGihAjwy', 'Lisa Park', 'Coffee enthusiast and home barista. Love sharing my coffee journey with others!', 'Beitou District, Taipei', true, NOW() - INTERVAL '3 months');

-- Create 50 diverse coffee listings
INSERT INTO listings (seller_id, title, description, category, price, quantity, pickup_instructions, expires_at, is_active, created_at) VALUES
    -- Sarah's listings (Light roast specialist)
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'Ethiopian Yirgacheffe - Washed Process', 'Bright, floral, and tea-like. Notes of lemon, bergamot, and jasmine. Roasted just 3 days ago. Perfect for pour-over brewing.', 'product', 32.00, 15, 'Ring buzzer 3B. Available weekday evenings after 6pm or weekends.', NOW() + INTERVAL '2 weeks', true, NOW() - INTERVAL '3 days'),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'Kenya AA - Nyeri Region', 'Juicy, wine-like acidity with black currant and tomato notes. Medium-light roast, excellent for V60.', 'product', 35.00, 10, 'Ring buzzer 3B. Available weekday evenings after 6pm or weekends.', NOW() + INTERVAL '2 weeks', true, NOW() - INTERVAL '5 days'),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'Colombian Geisha - Exclusive Micro-lot', 'Rare Geisha variety from Risaralda. Incredibly complex with notes of tropical fruit, honey, and florals.', 'product', 85.00, 5, 'Ring buzzer 3B. Special packaging for this exclusive lot.', NOW() + INTERVAL '10 days', true, NOW() - INTERVAL '1 day'),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'Pour-Over Masterclass', 'Learn my techniques for the perfect V60 brew. Includes tasting of 3 different origins. 90-minute session.', 'experience', 45.00, 4, 'My home studio in Xinyi. Address provided after booking.', NOW() + INTERVAL '1 month', true, NOW() - INTERVAL '7 days'),

    -- Mike's listings (Espresso specialist)
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12', 'House Espresso Blend - "Midnight Oil"', 'My signature blend: 60% Brazil, 30% Colombia, 10% Ethiopia. Chocolate, caramel, with bright fruit finish. Perfect for milk drinks.', 'product', 28.00, 20, 'Meet at lobby of my apartment building. Flexible timing on weekends.', NOW() + INTERVAL '3 weeks', true, NOW() - INTERVAL '2 days'),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12', 'Italian Roast - Traditional Blend', 'Dark and bold, just like nonna used to make. Notes of dark chocolate, smoke, and brown sugar.', 'product', 25.00, 18, 'Meet at lobby of my apartment building.', NOW() + INTERVAL '3 weeks', true, NOW() - INTERVAL '4 days'),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12', 'Espresso Tasting Experience', 'Pull shots on my La Marzocco Linea Mini! Try 5 different espresso profiles, learn about extraction theory.', 'experience', 55.00, 3, 'My home coffee bar in Da''an. Full address after confirmation.', NOW() + INTERVAL '2 weeks', true, NOW() - INTERVAL '6 days'),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12', 'Decaf Espresso - Swiss Water Process', 'Who says decaf can''t be delicious? Smooth, sweet, with notes of milk chocolate and hazelnut.', 'product', 30.00, 12, 'Lobby pickup, evenings preferred.', NOW() + INTERVAL '2 weeks', true, NOW() - INTERVAL '3 days'),

    -- Emma's listings (Direct trade specialist)
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13', 'Yunnan Single Origin - Honey Process', 'Direct from my partner farm in Pu''er. Honey process brings out incredible sweetness. Notes of brown sugar, orange, and honey.', 'product', 38.00, 15, 'Meet at Zhongshan MRT station exit 2, or pickup from my studio.', NOW() + INTERVAL '2 weeks', true, NOW() - INTERVAL '2 days'),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13', 'Taiwan High Mountain Oolong Coffee', 'Rare coffee grown in Alishan using tea cultivation methods. Unique floral and delicate profile.', 'product', 75.00, 8, 'Studio pickup in Zhongshan. By appointment only.', NOW() + INTERVAL '1 week', true, NOW() - INTERVAL '1 day'),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13', 'Brewing Methods Comparison Class', 'Compare V60, Chemex, Aeropress, and French Press. Same coffee, four methods. Eye-opening experience!', 'experience', 50.00, 6, 'My studio in Zhongshan. Includes all materials and take-home samples.', NOW() + INTERVAL '3 weeks', true, NOW() - INTERVAL '5 days'),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13', 'Guatemala Bourbon - Natural Process', 'Sweet and fruity natural process. Strawberry, chocolate, and wine notes. Great for cold brew too!', 'product', 34.00, 20, 'Flexible pickup at my studio or MRT station.', NOW() + INTERVAL '2 weeks', true, NOW() - INTERVAL '4 days'),

    -- David's listings (Professional roaster)
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a14', 'Roaster''s Choice Subscription Sample', 'Try 3 different roast profiles of the same bean. Understand how roasting affects flavor. Educational and delicious!', 'product', 42.00, 25, 'Pickup at my roastery in Songshan. Tours available upon request!', NOW() + INTERVAL '1 month', true, NOW() - INTERVAL '7 days'),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a14', 'Brazil Pulped Natural - Competition Grade', 'Cup of Excellence winner. Perfectly balanced with notes of chocolate, nuts, and subtle fruit.', 'product', 45.00, 30, 'Roastery pickup. Can arrange evening or weekend times.', NOW() + INTERVAL '3 weeks', true, NOW() - INTERVAL '3 days'),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a14', 'Roasting Workshop - Beginner Friendly', 'Learn to roast your own coffee! Small group workshop at my roastery. Includes 1kg of green beans to take home.', 'experience', 120.00, 2, 'My roastery in Songshan. Full day workshop 10am-4pm.', NOW() + INTERVAL '1 month', true, NOW() - INTERVAL '10 days'),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a14', 'Aged Sumatra - 3 Year Monsooned', 'Unique aged coffee with earthy, herbal, and tobacco notes. An acquired taste for adventurous palates!', 'product', 40.00, 10, 'Roastery pickup. Try before you buy!', NOW() + INTERVAL '1 month', true, NOW() - INTERVAL '2 days'),

    -- Lisa's listings (Home barista)
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a15', 'Beginner''s Coffee Tasting Set', 'Perfect for those starting their specialty coffee journey. 5 different origins, 100g each, with brewing guide.', 'product', 55.00, 8, 'Meet at Beitou MRT or home pickup on weekends.', NOW() + INTERVAL '2 weeks', true, NOW() - INTERVAL '4 days'),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a15', 'Latte Art Practice Session', 'Casual session to practice latte art together. Bring your own milk! I provide the espresso and guidance.', 'experience', 25.00, 4, 'My apartment in Beitou. Relaxed and fun atmosphere!', NOW() + INTERVAL '2 weeks', true, NOW() - INTERVAL '6 days'),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a15', 'Cold Brew Concentrate - Ready to Drink', 'My special cold brew blend. 1 liter bottle, just dilute and enjoy. Smooth, chocolatey, never bitter.', 'product', 22.00, 15, 'Pickup from Beitou MRT station. Keep refrigerated!', NOW() + INTERVAL '1 week', true, NOW() - INTERVAL '2 days'),

    -- More varied listings for rich content
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'Panama Geisha - Auction Lot', 'From Hacienda La Esmeralda. Jasmine, tropical fruits, and champagne-like. The pinnacle of coffee.', 'product', 150.00, 3, 'Special packaging. By appointment only.', NOW() + INTERVAL '5 days', true, NOW() - INTERVAL '1 day'),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12', 'Cappuccino Perfection Workshop', 'Master the perfect cappuccino. Milk steaming, pouring techniques, and the golden ratio.', 'experience', 40.00, 5, 'My coffee bar. Morning sessions only.', NOW() + INTERVAL '2 weeks', true, NOW() - INTERVAL '3 days'),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13', 'Indonesian Sumatra - Wet Hulled', 'Full body, low acidity, herbal and earthy notes. Perfect for French Press lovers.', 'product', 30.00, 22, 'Studio pickup or MRT meetup.', NOW() + INTERVAL '3 weeks', true, NOW() - INTERVAL '5 days'),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a14', 'Coffee Cupping Session', 'Professional cupping session. Learn to taste coffee like a Q-grader. Educational and fun!', 'experience', 35.00, 8, 'Roastery location. Includes cupping materials.', NOW() + INTERVAL '2 weeks', true, NOW() - INTERVAL '4 days'),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a15', 'Japanese Style Iced Coffee', 'Flash-chilled brewing method. Bright, clean, refreshing. Perfect for Taiwan summers!', 'product', 26.00, 18, 'Beitou area pickup.', NOW() + INTERVAL '2 weeks', true, NOW() - INTERVAL '3 days'),

    -- Additional diverse listings
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'Costa Rica Red Honey Process', 'Sweet and syrupy, with notes of red apple, brown sugar, and mild citrus. Medium-light roast.', 'product', 36.00, 14, 'Xinyi pickup, flexible times.', NOW() + INTERVAL '2 weeks', true, NOW() - INTERVAL '2 days'),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12', 'Espresso Machine Maintenance Class', 'Learn to maintain your home espresso machine. Cleaning, descaling, and basic repairs.', 'experience', 60.00, 3, 'Da''an workshop space.', NOW() + INTERVAL '3 weeks', true, NOW() - INTERVAL '8 days'),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13', 'Rwanda Bourbon - Women''s Cooperative', 'Supporting women farmers. Clean, sweet, with notes of red grape and brown sugar.', 'product', 37.00, 16, 'Zhongshan district pickup.', NOW() + INTERVAL '2 weeks', true, NOW() - INTERVAL '3 days'),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a14', 'Green Coffee Beans - DIY Roasting', 'High-quality green beans for home roasting. 1kg bags, multiple origins available.', 'product', 20.00, 40, 'Roastery pickup anytime.', NOW() + INTERVAL '2 months', true, NOW() - INTERVAL '5 days'),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a15', 'Coffee and Dessert Pairing', 'Learn to pair coffee with desserts. Tasting session includes 4 pairings. Sweet tooth heaven!', 'experience', 38.00, 6, 'Beitou location, weekend afternoons.', NOW() + INTERVAL '2 weeks', true, NOW() - INTERVAL '4 days'),

    -- Premium and unique offerings
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'Jamaica Blue Mountain - Certified', 'The legendary Blue Mountain. Mild, sweet, and incredibly smooth. Limited quantity.', 'product', 120.00, 5, 'By appointment only. Certificate included.', NOW() + INTERVAL '1 week', true, NOW() - INTERVAL '2 days'),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12', 'Home Barista Setup Consultation', 'I''ll help you design your perfect home coffee setup. Equipment recommendations and setup assistance.', 'experience', 80.00, 2, 'Can visit your home or meet online.', NOW() + INTERVAL '1 month', true, NOW() - INTERVAL '6 days'),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13', 'Cascara Tea - Coffee Cherry Tea', 'Made from coffee cherry husks. Fruity, sweet, naturally caffeinated. Unique and sustainable!', 'product', 18.00, 25, 'Studio pickup, try before you buy!', NOW() + INTERVAL '3 weeks', true, NOW() - INTERVAL '3 days'),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a14', 'Kopi Luwak - Ethically Sourced', 'The famous civet coffee, ethically sourced from wild civets. Smooth, less acidic, unique flavor.', 'product', 200.00, 2, 'Special order, serious inquiries only.', NOW() + INTERVAL '2 weeks', true, NOW() - INTERVAL '1 day'),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a15', 'Coffee Cocktails Workshop', 'Learn to make espresso martinis, cold brew cocktails, and coffee-infused spirits!', 'experience', 65.00, 4, 'Evening sessions only, 21+ required.', NOW() + INTERVAL '2 weeks', true, NOW() - INTERVAL '5 days'),

    -- Budget-friendly options
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'Sample Pack - Light Roast Trio', 'Can''t decide? Try 3 different light roasts, 50g each. Perfect for exploration!', 'product', 18.00, 30, 'Quick pickup available daily.', NOW() + INTERVAL '1 month', true, NOW() - INTERVAL '4 days'),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12', 'Second Crack Special - Dark Roast', 'My experimental dark roast. Bold, intense, slightly smoky. Great price for adventurous drinkers!', 'product', 20.00, 35, 'Lobby pickup anytime.', NOW() + INTERVAL '1 month', true, NOW() - INTERVAL '7 days'),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13', 'Filter Coffee Basics', 'Introduction to filter coffee. Learn the fundamentals in a relaxed, small group setting.', 'experience', 30.00, 8, 'Weekend mornings at my studio.', NOW() + INTERVAL '3 weeks', true, NOW() - INTERVAL '6 days'),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a14', 'Roast Date Special - 7 Days Old', 'Perfectly rested coffee at a discount. Still fresh, just right for brewing!', 'product', 22.00, 50, 'Large quantity available!', NOW() + INTERVAL '1 week', true, NOW() - INTERVAL '3 days'),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a15', 'Coffee Grounds for Gardening', 'Used coffee grounds, perfect for composting or gardening. Free, just bring a container!', 'product', 0.00, 100, 'Available anytime, just message first.', NOW() + INTERVAL '1 month', true, NOW() - INTERVAL '2 days'),

    -- Seasonal and limited editions
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'Christmas Blend 2024', 'Festive blend with notes of cinnamon, orange peel, and chocolate. Limited holiday edition!', 'product', 35.00, 20, 'Gift wrapping available.', NOW() + INTERVAL '3 weeks', true, NOW() - INTERVAL '3 days'),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12', 'Valentine''s Day Couples Brewing', 'Romantic coffee experience for two. Learn to brew each other''s perfect cup!', 'experience', 75.00, 3, 'Intimate setting, reservation required.', NOW() + INTERVAL '1 month', true, NOW() - INTERVAL '5 days'),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13', 'Lunar New Year Special Blend', 'Prosperity blend with lucky red packaging. Sweet, balanced, perfect for gifting.', 'product', 38.00, 28, 'Special CNY packaging included.', NOW() + INTERVAL '2 weeks', true, NOW() - INTERVAL '4 days'),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a14', 'Summer Cold Brew Pack', 'Everything you need for cold brew season. Beans, brewing bag, and instructions.', 'product', 32.00, 25, 'Get ready for summer!', NOW() + INTERVAL '1 month', true, NOW() - INTERVAL '6 days'),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a15', 'Autumn Harvest Blend', 'Seasonal blend with notes of apple, caramel, and spices. Cozy autumn vibes!', 'product', 33.00, 18, 'Limited autumn edition.', NOW() + INTERVAL '3 weeks', true, NOW() - INTERVAL '3 days'),

    -- Final listings to reach 50
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'Ethiopian Natural Process - Fruity Bomb', 'Explosion of fruit flavors! Blueberry, strawberry, tropical fruits. Not for the faint-hearted!', 'product', 40.00, 12, 'Xinyi district, evening pickup.', NOW() + INTERVAL '2 weeks', true, NOW() - INTERVAL '2 days'),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12', 'Flat White Focus Session', 'Perfect your flat white skills. Microfoam techniques and the Australian way.', 'experience', 42.00, 4, 'Morning sessions at my coffee bar.', NOW() + INTERVAL '2 weeks', true, NOW() - INTERVAL '4 days'),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13', 'Vietnamese Robusta - Traditional Style', 'Bold, strong, perfect for Vietnamese coffee. Includes free phin filter with purchase!', 'product', 24.00, 20, 'Zhongshan pickup, filter included.', NOW() + INTERVAL '1 month', true, NOW() - INTERVAL '5 days'),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a14', 'Coffee Storage Workshop', 'Learn proper storage techniques to keep your coffee fresh longer. Practical tips and demonstrations.', 'experience', 28.00, 10, 'Roastery workshop space.', NOW() + INTERVAL '3 weeks', true, NOW() - INTERVAL '7 days');

-- Add some test orders to show activity
INSERT INTO orders (listing_id, buyer_id, seller_id, quantity, amount, state, created_at)
SELECT
    l.id,
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a15', -- Lisa as buyer
    l.seller_id,
    1,
    l.price,
    'completed',
    NOW() - INTERVAL '2 weeks'
FROM listings l
LIMIT 3;

-- Add some reviews for completed orders
INSERT INTO reviews (order_id, reviewer_id, rating, comment, created_at)
SELECT
    o.id,
    o.buyer_id,
    4 + (RANDOM())::int,  -- This gives 4 or 5
    'Great coffee and smooth transaction!',
    NOW() - INTERVAL '1 week'
FROM orders o
WHERE o.state = 'completed'
LIMIT 3;