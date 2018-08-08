#!/bin/sh

set -o errexit
set -o pipefail

VENDOR=v3io
DRIVER=fuse

# Assuming the single driver file is located at /$DRIVER inside the DaemonSet image.


driver_dir=$VENDOR${VENDOR:+"~"}${DRIVER}
if [ ! -d "/flexmnt/$driver_dir" ]; then
  mkdir "/flexmnt/$driver_dir"
fi

cp "/$DRIVER" "/flexmnt/$driver_dir/.$DRIVER"
mv -f "/flexmnt/$driver_dir/.$DRIVER" "/flexmnt/$driver_dir/$DRIVER"

cp "/etc/config/v3io/v3io.conf" "/etc/v3io/fuse/v3io.conf"

while : ; do
  sleep 3600
done