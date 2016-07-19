etcdbot
================

Sorry! Only Japanese document is available.

etcdの特定のkeyをwatchするwatcherと、管理用botです。


Installation
------------

    go get github.com/okzk/etcdbot/etcdwatcher
    go get github.com/okzk/etcdbot

Usage
-----

以下の環境変数を設定してください

- BOT_ETCD_ENDPOINTS
  - etcのエンドポイントをカンマ区切りで
  - デフォルト "http://localhost:2379"
- BOT_ETCD_USER
  - etcdのユーザ(optional)
- BOT_ETCD_PASSWORD
  - etcdのパスワード(optional)
- BOT_METADATA_DIR
  - etcd上でbotが使うデータ保存用ディレクトリ
  - デフォルト "/etcdbot_meta"
- BOT_SLACK_API_KEY
  - slack APIのkey(etcdbotではrequired)

etcdで以下のkeyに値を設定してください。オンラインで設定変更可能です。

- $BOT_METADATA_DIR/incomingWebHookUrls
  - 通知先のincomingWebHookのURLリストをカンマ区切りで
- $BOT_METADATA_DIR/watchTargetList
  - watch対象のパスのリストをカンマ区切りで


[seelog](https://github.com/cihub/seelog)のxmlファイルを指定することでログ出力を調整することもできます。

    etcdbot -cfg config.yml -log log.xml


License
-------

MIT
