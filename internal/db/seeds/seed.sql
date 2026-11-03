INSERT INTO hotels (name)
VALUES ('Grand Plaza Hotel'), ('Seaside Inn')
ON CONFLICT DO NOTHING;