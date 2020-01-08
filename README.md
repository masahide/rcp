# rcp
remote copy

![screenshot](./screenshot.png)

Overview
----

Commands for file transfer by tcp


Characteristic
----

- Transfer files using buffer between file read / write and transfer process
- Monitor read / write speed and transfer speed every second and display them in [Sparkline chart](https://images.app.goo.gl/f2adCQNKfWCG4ZJV8)
- Network and storage performance can be measured with dummy data transmission and dummy reception functions

download
-----------

[Release page](https://github.com/masahide/rcp/releases) Select and download the one that matches your platform

How to use
---------

The main procedure is performed in the following two steps

- Listen to any port number on the receiving side
- Dial the destination port number on the sending side


Usage of listen mode
---------------------

```
Usage:
  rcp listen [flags]

Flags:
  -h, --help                help for listen
  -l, --listenAddr string   listen address (default "0.0.0.0:1987")
  -o, --output string       output filename

Global Flags:
      --bufSize int         Buffer size (default 10485760)
      --dummyInput string   dummy input mode data size (ex: 100MB, 4K, 10g)
      --dummyOutput         dummy output mode
      --maxBufNum int       Maximum number of buffers (default 100)
```


Usage of send mode
---------------------

```
Usage:
  rcp send [flags]

Flags:
  -d, --dialAddr string   dial address (ex: "198.51.100.1:1987" )
  -h, --help              help for send
  -i, --input string      input filename

Global Flags:
      --bufSize int         Buffer size (default 10485760)
      --dummyInput string   dummy input mode data size (ex: 100MB, 4K, 10g)
      --dummyOutput         dummy output mode
      --maxBufNum int       Maximum number of buffers (default 100)
```



Example of use
-----

### When listening on 1987 port on the receiving side (IP: 10.10.10.10) and sending

- Listen on TCP `1987` port on the receiving side

```bash
$ rcp listen -l: 1987 -o save_filename
```

- Send file to `10.10.10.10: 1987`

```bash
$ rcp send -d 10.10.10.10:1987 -i input_filename
```

### Dummy data transmission-> Discard received dummy data

- Receiver
```
$ rcp listen -l: 1987 --dummyOutput
```
- Sender
```bash
$ rcp send -d 10.10.10.10:1987 -i input_filename
```
