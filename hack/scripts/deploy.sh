#!/bin/sh

set -o errexit
set -o pipefail

VENDOR=v3io
DRIVER=fuse

driver_dir=$VENDOR${VENDOR:+"~"}${DRIVER}
echo "$(date) - Preparing to install $driver_dir"

plugin_dir="/flexmnt/$driver_dir"
if [ -d "$plugin_dir" ]; then
  echo "$(date) - Driver exists at $plugin_dir - replacing"
  rm -rf "$plugin_dir"
fi

install_dir="/flexmnt/.$driver_dir"
if [ ! -d "$install_dir" ]; then
  echo "$(date) - Creating temp installation folder in $install_dir"
  mkdir "$install_dir"
fi

echo "$(date) - Copying $DRIVER to $install_dir/$DRIVER"
cp "/$DRIVER" "$install_dir/$DRIVER"

echo "$(date) - Copying config from '/etc/config/v3io/v3io.conf' to '/etc/v3io/fuse/v3io.conf'"
cp "/etc/config/v3io/v3io.conf" "/etc/v3io/fuse/v3io.conf"
echo "-------------- v3io.conf BEGIN ----------------"
cat /etc/v3io/fuse/v3io.conf
echo "-------------- v3io.conf END   ----------------"

echo "$(date) - Copying install.sh and libs folder to $install_dir"
cp "/install.sh" "$install_dir/"
cp -r "/libs" "$install_dir/"

ls -lahR "$install_dir"

# Signal fsnotify (This must be the last operation before going to sleep)
echo "$(date) - Moving $install_dir to $plugin_dir"
mv -f "$install_dir" "$plugin_dir"

echo "$(date) - Completed. Going to sleep now"
while : ; do
  sleep 3600
done