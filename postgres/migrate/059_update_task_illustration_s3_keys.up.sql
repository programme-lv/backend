UPDATE tasks
SET illustr_img_s3_key = new_images.illustr_img_s3_key
FROM (
    VALUES
        ('paklajs', 'paklajs.63bfcd42.jpg'),
        ('bulcinas', 'bulcinas.91283e96.png'),
        ('monetas', 'monetas.1788cf7c.png'),
        ('siers', 'siers.a8f24b7d.jpg'),
        ('netpilsetas', 'netpilsetas.3bca9035.png'),
        ('tornisdivi', 'tornisdivi.f18b9877.png'),
        ('vitrina', 'vitrina.d8163c15.png'),
        ('uzmini', 'uzmini.e38b632c.png'),
        ('konfektes', 'konfektes.f770fc1c.jpg'),
        ('otraiscel', 'otraiscel.eb96eaf7.jpg'),
        ('seifs', 'seifs.973727e5.png'),
        ('meli', 'meli.e591528e.png'),
        ('kvadrputekl', 'kvadrputekl.bafcc0aa.jpg'),
        ('hokejs', 'hokejs.8d772b35.png'),
        ('adapteri', 'adapteri.204a9981.jpg')
) AS new_images(short_id, illustr_img_s3_key)
WHERE tasks.short_id = new_images.short_id;
