#!/usr/bin/env bash
# Copyright 2018 Iguazio
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

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
    echo "$(date) - Installing v3io-fuse package using 'rpm -ivh --force ${LIBS_DIR}/${PACKAGE_NAME}.rpm'" >> /tmp/init.log
    rpm -ivh --force ${LIBS_DIR}/${PACKAGE_NAME}.rpm &>> /tmp/init.log
else
    echo "$(date) - Installing v3io-fuse package using 'dpkg -i ${LIBS_DIR}/${PACKAGE_NAME}.deb'" >> /tmp/init.log
    dpkg -i ${LIBS_DIR}/${PACKAGE_NAME}.deb &>> /tmp/init.log
fi

echo "$(date) - Installation Completed" >> /tmp/init.log