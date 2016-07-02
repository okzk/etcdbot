etcdbot
================

Sorry! Only Japanese document is available.

etcdの特定のkeyをwatchして、その内容をslackに通知するbotです。
設定はsample_config.ymlを見てください


Installation
------------

    go get github.com/okzk/etcdbot

Usage
-----

yamlの設定ファイルを指定して起動してください

    etcdbot -cfg config.yml

[seelog](https://github.com/cihub/seelog)のxmlファイルを指定することでログ出力を調整することもできます。

    etcdbot -cfg config.yml -log log.xml


License
-------

MIT
