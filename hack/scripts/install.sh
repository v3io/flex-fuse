#!/usr/bin/env bash

set -e

LIBS_DIR="$(dirname "$0")/libs"
PKG_MANAGER=""
PACKAGES="fuse librdmacm"
PACKAGE_NAME="igz-fuse"

echo "$(date) - checking yum or apt" >> /tmp/init.log
HAS_YUM=$(which yum &> /dev/null; echo $?)
HAS_APT=$(which apt &> /dev/null; echo $?)

if [ "${HAS_YUM}" == "0" ]; then
    PKG_MANAGER="yum"
elif [ "${HAS_APT}" == "0" ]; then
    PKG_MANAGER="apt-get"
    PACKAGES="fuse librdmacm1"
    apt-get update
else
    echo "Installation supports 'yum' or 'apt-get'"
    exit 1
fi

echo "$(date) - PKG_MANAGER is ${PKG_MANAGER}" >> /tmp/init.log

echo "Installing required packages"
echo "$(date) - Installing required packages - ${PACKAGES}" >> /tmp/init.log

${PKG_MANAGER} install -y ${PACKAGES} &>> /tmp/init.log

echo "Installing v3io-fuse package"
echo "$(date) - Installing v3io-fuse package" >> /tmp/init.log

if [ "${HAS_YUM}" == "0" ]; then
    echo "$(date) - Installing v3io-fuse package using 'rpm -ivh ${LIBS_DIR}/${PACKAGE_NAME}.rpm'" >> /tmp/init.log
    rpm -ivh ${LIBS_DIR}/${PACKAGE_NAME}.rpm &>> /tmp/init.log
else
    echo "$(date) - Installing v3io-fuse package using 'dpkg -i ${LIBS_DIR}/${PACKAGE_NAME}.deb'" >> /tmp/init.log
    dpkg -i ${LIBS_DIR}/${PACKAGE_NAME}.deb &>> /tmp/init.log
fi

echo "$(date) - Installation Completed" >> /tmp/init.log