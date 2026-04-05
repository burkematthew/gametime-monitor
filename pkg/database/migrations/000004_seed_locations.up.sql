INSERT INTO locations (name, address) VALUES
    ('Woodlands', '1 Woodlands Parkway, St Peters, MO 63376'),
    ('Fenton', '945 Larkin Williams Rd, Fenton, MO 63026'),
    ('Strehl Fields', '6000 State Hwy N, Cottleville, MO 63304'),
    ('Fenton City Park', '1215 Larkin Williams Rd, Fenton, MO 63026'),
    ('Affton Athletic Association', '10300 Gravois Rd, St. Louis, MO'),
    ('Northwest Athletic', '6098 Country Creek Dr, House Springs, MO 63051'),
    ('C & H Ballpark', '2780 St Peters Howell Rd, St Peters, MO 63376'),
    ('Old Town St Peters', '1 Park St, St Peters, MO 63376')
ON CONFLICT (name) DO NOTHING;
