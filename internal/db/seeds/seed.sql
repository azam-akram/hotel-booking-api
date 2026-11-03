-- ============================================
-- Hotel Booking System - Seed Data
-- ============================================

BEGIN;

-- ============================================
-- 1. HOTELS
-- ============================================

INSERT INTO hotels (name)
VALUES
    ('Grand Palace Hotel'),
    ('Ocean View Resort'),
    ('Mountain Escape Lodge')
ON CONFLICT DO NOTHING;


-- ============================================
-- 2. ROOMS
-- Exactly 6 rooms per hotel
-- ============================================

INSERT INTO rooms (hotel_id, number, type, capacity)
SELECT
    h.id,
    r.number,
    r.type::room_type,
    r.capacity
FROM hotels h
CROSS JOIN (
    VALUES
        ('101', 'single', 1),
        ('102', 'single', 1),
        ('201', 'double', 2),
        ('202', 'double', 2),
        ('301', 'deluxe', 3),
        ('302', 'deluxe', 4)
) AS r(number, type, capacity)
WHERE h.name IN (
    'Grand Palace Hotel',
    'Ocean View Resort',
    'Mountain Escape Lodge'
)
ON CONFLICT (hotel_id, number) DO NOTHING;


-- ============================================
-- 3. BOOKINGS
-- All bookings respect room capacity.
-- No bookings overlap in the same room.
-- ============================================

INSERT INTO bookings (
    reference,
    room_id,
    guest_name,
    guest_email,
    check_in,
    check_out,
    guests
)
SELECT
    b.reference,
    r.id,
    b.guest_name,
    b.guest_email,
    b.check_in::DATE,
    b.check_out::DATE,
    b.guests
FROM (
    VALUES
        -- Grand Palace Hotel
        (
            'BK-0001', 'Grand Palace Hotel', '101',
            'John Smith', 'john@example.com',
            '2026-11-05', '2026-11-08', 1
        ),
        (
            'BK-0002', 'Grand Palace Hotel', '201',
            'Sarah Johnson', 'sarah@example.com',
            '2026-11-05', '2026-11-10', 2
        ),
        (
            'BK-0003', 'Grand Palace Hotel', '301',
            'Michael Brown', 'michael@example.com',
            '2026-11-12', '2026-11-16', 3
        ),
        (
            'BK-0004', 'Grand Palace Hotel', '101',
            'David Wilson', 'david@example.com',
            '2026-11-08', '2026-11-12', 1
        ),

        -- Ocean View Resort
        (
            'BK-0005', 'Ocean View Resort', '102',
            'Emily Davis', 'emily@example.com',
            '2026-11-06', '2026-11-09', 1
        ),
        (
            'BK-0006', 'Ocean View Resort', '301',
            'Robert Miller', 'robert@example.com',
            '2026-11-10', '2026-11-15', 3
        ),
        (
            'BK-0007', 'Ocean View Resort', '202',
            'Jessica Taylor', 'jessica@example.com',
            '2026-11-15', '2026-11-18', 2
        ),

        -- Mountain Escape Lodge
        (
            'BK-0008', 'Mountain Escape Lodge', '201',
            'Daniel Anderson', 'daniel@example.com',
            '2026-11-07', '2026-11-11', 2
        ),
        (
            'BK-0009', 'Mountain Escape Lodge', '302',
            'Sophia Thomas', 'sophia@example.com',
            '2026-11-12', '2026-11-17', 4
        ),
        (
            'BK-0010', 'Mountain Escape Lodge', '101',
            'William Jackson', 'william@example.com',
            '2026-11-17', '2026-11-20', 1
        )
) AS b(
    reference,
    hotel_name,
    room_number,
    guest_name,
    guest_email,
    check_in,
    check_out,
    guests
)
JOIN hotels h
    ON h.name = b.hotel_name
JOIN rooms r
    ON r.hotel_id = h.id
    AND r.number = b.room_number
WHERE b.guests <= r.capacity
ON CONFLICT (reference) DO NOTHING;

COMMIT;
