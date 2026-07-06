UPDATE task_images
SET s3_key = new_images.s3_key
FROM (
    VALUES
        ('adapteri', '1.png', 'adapteri/c47dcad1a16e.png'),
        ('atklatrutinas', '1.png', 'atklatrutinas/aa0503adedb3.png'),
        ('atklatrutinas', '2.png', 'atklatrutinas/fad82a2829b2.png'),
        ('hokejs', '1.png', 'hokejs/955a61a9ca91.png'),
        ('kvadrputekl', '1.png', 'kvadrputekl/20d89694620a.png'),
        ('kvadrputekl', '2.png', 'kvadrputekl/765971923f4b.png'),
        ('netpilsetas', '1.png', 'netpilsetas/46d5f34a6558.png'),
        ('otraiscel', '1.png', 'otraiscel/3836d280ff42.png'),
        ('pbumbinas', '1.png', 'pbumbinas/5acbcf7576df.png'),
        ('pbumbinas', '2.png', 'pbumbinas/712f026732ca.png'),
        ('seifs', '1.png', 'seifs/d4ebe52d1b49.png'),
        ('tornisdivi', '1.png', 'tornisdivi/d9e5518af2be.png'),
        ('bojatais', 'bojatais.png', 'bojatais/1a9893c00707.png'),
        ('dargakais', 'dargakais.png', 'dargakais/b202b08c1944.png'),
        ('dargakais', 'dargakais2.png', 'dargakais/847458662223.png'),
        ('paklajs', 'paklajs_0.png', 'paklajs/94ea4a320548.png'),
        ('paklajs', 'paklajs_0a.png', 'paklajs/290f7ac3ac45.png'),
        ('paklajs', 'paklajs_1.png', 'paklajs/22ee8be14804.png'),
        ('paklajs', 'paklajs_1a.png', 'paklajs/78a725846f85.png'),
        ('paklajs', 'paklajs_2.png', 'paklajs/d41c0718c355.png'),
        ('paklajs', 'paklajs_2a.png', 'paklajs/8ef45085e8c8.png'),
        ('ielas', 'ielas.png', 'ielas/9d1f90ec7292.png'),
        ('ielas', 'puses.png', 'ielas/438c91e15d7a.png')
) AS new_images(task_short_id, file_name, s3_key)
WHERE task_images.task_short_id = new_images.task_short_id
  AND task_images.file_name = new_images.file_name;
