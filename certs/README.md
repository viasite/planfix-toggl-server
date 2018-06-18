Сертификаты для localhost сгенерированы командой по [статье](https://letsencrypt.org/docs/certificates-for-localhost/):

```
openssl req -x509 -days 1825 -out server.crt -keyout server.key \
  -newkey rsa:2048 -nodes -sha256 \
  -subj '/CN=localhost' -extensions EXT -config <( \
   printf "[dn]\nCN=localhost\n[req]\ndistinguished_name = dn\n[EXT]\nsubjectAltName=DNS:localhost\nkeyUsage=digitalSignature\nextendedKeyUsage=serverAuth")
```

Чтобы браузер не ругался на самоподписанные сертификаты, нужно установить сертификат в систему.

## Установка сертификата на Windows
- 2 раза кликнуть по сертификату с иконкой (server.crt)
- Установить сертификат
- Локальный компьютер
- Поместить все сертификаты в следующее хранилище (Обзор)
- Поставить галочку "Показать физические зхранилища"
- Доверенные корневые центры сертификации - Реестр
