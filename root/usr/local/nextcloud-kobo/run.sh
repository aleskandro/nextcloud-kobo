#!/bin/sh

if pgrep "nextcloud-kobo"; then
    echo "nextcloud-kobo is already running"
    exit 0
fi

mkdir -p /mnt/onboard/.adds/nextcloud-kobo
mkdir -p /mnt/onboard/nextcloud
cp /usr/local/nextcloud-kobo/config.example.yaml /mnt/onboard/.adds/nextcloud-kobo/config.example.yaml

if [ ! -f /mnt/onboard/.adds/nextcloud-kobo/config.yaml ]; then
    echo "Configuration file not found. Do not run."
    qndb -m mwcToast 5000 "NextCloud-Kobo" "Configuration file not found. Not running."
    exit 0
fi

# Clean the log file if the size is greater than 2MB
if [ -f /mnt/onboard/.adds/nextcloud-kobo/nextcloud-kobo.log ] && \
  [ "$(stat -c %s /mnt/onboard/.adds/nextcloud-kobo/nextcloud-kobo.log)" -gt 2097152 ]; then
        echo "Log file is greater than 2MB. Cleaning it."
        echo "" > /mnt/onboard/.adds/nextcloud-kobo/nextcloud-kobo.log
fi
(while true; do
/usr/local/nextcloud-kobo/nextcloud-kobo -config-file /mnt/onboard/.adds/nextcloud-kobo/config.yaml \
  -base-path /mnt/onboard/nextcloud >> /mnt/onboard/.adds/nextcloud-kobo/nextcloud-kobo.log 2>&1
done) &