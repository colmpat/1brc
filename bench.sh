#!/usr/bin/env fish
set buf (math "2^24")
for workers in (seq 8 15)
  set workers (math "2*$workers")
  echo "BUFFLEN=$buf WORKERS=$workers"
  BUFFLEN=$buf WORKERS=$workers ./1brc data/measurements.txt 1>/dev/null 2>> bench.log
end
