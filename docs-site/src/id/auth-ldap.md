# Autentikasi LDAP / AD

> English version: [LDAP / AD Authentication](../en/auth-ldap.md)

MWX-ISP dapat memverifikasi password tertentu terhadap direktori LDAP/AD.
Mode bind, TLS, batasan metode, penanganan koneksi (satu koneksi per upaya
autentikasi, tanpa pool, dengan `ldap.Timeout` membatasi seluruh pertukaran ke
directory — bukan tiap round trip), dan konfigurasi rinci dijelaskan dalam
[panduan English](../en/auth-ldap.md).
