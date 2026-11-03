-- Removes all data and restarts ids, ready for seeding. The schema is kept.
TRUNCATE bookings, rooms, hotels RESTART IDENTITY;
