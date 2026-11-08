CREATE TABLE movements (
    id      text PRIMARY KEY,
    warehouse_id text NOT NULL,
    code    text NOT NULL,
    label   text NOT NULL
);

INSERT INTO movements (id, warehouse_id, code, label)
SELECT 'MV-' || lpad(n::text, 6, '0'), 'WH-1', 'C-' || n, 'Mouvement ' || n
FROM generate_series(1, 1231) AS n;

INSERT INTO movements (id, warehouse_id, code, label) VALUES
    ('MV-100001', 'WH-1', 'C-Q', 'Palette "urgente"'),
    ('MV-100002', 'WH-1', 'C-B', 'Quai C:\nord'),
    ('MV-100003', 'WH-1', 'C-N', 'Ligne 1' || chr(10) || 'Ligne 2');

INSERT INTO movements (id, warehouse_id, code, label)
SELECT 'MV-' || lpad((200000 + n)::text, 6, '0'), 'WH-2', 'C-' || n, 'Autre entrepôt ' || n
FROM generate_series(1, 10) AS n;
