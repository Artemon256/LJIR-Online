# LJIR-Online
LiveJournal Image Reuploader Online

https://ljir.devnullinc.pp.ua/

You can and should modify ljir.conf before using:


site_tls: boolean, determines whenever site should use TLS (SSL) or not. Default: false

site_cert: path to SSL certificate file (works only if site_tls is true)

site_key: path to SSL key file (works only if site_tls is true)


imgur_clientID: ClientID of your Imgur application

imgur_clientSecret: ClientSecret of your Imgur application

imgur_mashapeKey: Mashape key of your Imgur application


smtp_username: username on your SMTP server

smtp_password: username's password on your SMTP server

smtp_server: domain or IP of your SMTP server


gid: id of group which files created by programs will belong to. Default: gid of user's group

uid: id of user which will own files created by programs. Default: uid of user
