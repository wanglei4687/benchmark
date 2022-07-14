#!/bin/bash

wget https://github.com/wanglei4687/fsyncperf/blob/main/bin/fsyncpref
chmod 755 fsyncpref
./fsyncpref --path $1 > index.html
nohup python -m SimpleHTTPServer 80 &
