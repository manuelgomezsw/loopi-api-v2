INSERT INTO unidades_medida
  (codigo, nombre, tipo_medida, factor_conversion, unidad_base, activo, creado_en, actualizado_en)
VALUES
  ('g',   'Gramo',     'peso',    1.0000, 1, 1, NOW(), NOW()),
  ('ml',  'Mililitro', 'volumen', 1.0000, 1, 1, NOW(), NOW()),
  ('und', 'Unidad',    'unidad',  1.0000, 1, 1, NOW(), NOW());

INSERT INTO unidades_medida
  (codigo, nombre, tipo_medida, factor_conversion, unidad_base, activo, creado_en, actualizado_en)
VALUES
  ('kg',     'Kilogramo',  'peso',    1000.0000,    0, 1, NOW(), NOW()),
  ('t',      'Tonelada',   'peso',    1000000.0000, 0, 1, NOW(), NOW()),
  ('mg',     'Miligramo',  'peso',    0.0010,       0, 1, NOW(), NOW()),
  ('L',      'Litro',      'volumen', 1000.0000,    0, 1, NOW(), NOW()),
  ('dL',     'Decilitro',  'volumen', 100.0000,     0, 1, NOW(), NOW()),
  ('cL',     'Centilitro', 'volumen', 10.0000,      0, 1, NOW(), NOW()),
  ('docena', 'Docena',     'unidad',  12.0000,      0, 1, NOW(), NOW()),
  ('par',    'Par',        'unidad',  2.0000,       0, 1, NOW(), NOW()),
  ('caja',   'Caja',       'unidad',  24.0000,      0, 1, NOW(), NOW());
