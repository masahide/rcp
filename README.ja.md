rcp
===
remote copy

![screenshot](./screenshot.png)



概要
----

tcpによるファイル転送用のコマンド


特徴
----

- ファイル読み書きと転送処理の間にバッファを用いてファイルを転送する
- 読み書き速度と転送速度を毎秒モニタリングしSparkline chartで表示する
- ダミーデータ送信やダミー受信機能でネットワークおよびストレージ性能を計測可能


ダウンロード
-----------

[リリースページ](https://github.com/masahide/rcp/releases) からプラットフォームにあった物を選んでダウンロード


使い方概要
---------

主な手順は以下の2つの手順で行う

- 受信側で任意のポート番号listenする
- 送信側で受信先のポート番号にdialする


使用例
-----

### 受信側(IP:10.10.10.10)で1987ポートでlistenして送信する場合

- 受信側でTCP `1987` port で listen

```bash
$ rcp listen -l :1987 -o save_filename
```

- 送信から `10.10.10.10:1987` へファイル送信

```bash
$ rcp send -d 10.10.10.10:1987 -i input_filename
```

### ダミーデータ送信 -> 受信ダミーデータを捨てる

- 受信側
```
$ rcp listen -l :1987 --dummyOutput
```
- 送信側
```bash
$ rcp send -d 10.10.10.10:1987 -i input_filename
```
