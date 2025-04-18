hwclock -s
apk update
apk add --no-cache zfs zfs-lts

/sbin/modprobe zfs

echo "https://dl-cdn.alpinelinux.org/alpine/v3.16/community/" >> /etc/apk/repositories

apk update

apk add --no-cache lxd lxd-client dbus

echo "session optional pam_cgfs.so -c freezer,memory,name=systemd,unified" >> /etc/pam.d/system-login
echo "lxc.idmap = u 0 100000 65536" >> /etc/lxc/default.conf
echo "lxc.idmap = g 0 100000 65536" >> /etc/lxc/default.conf
echo "root:100000:65536" >> /etc/subuid
echo "root:100000:65536" >> /etc/subgid

# If you plan to run systemd based Linux distributions (Debian, Ubuntu, etc.)
echo "systemd_container=yes" >>  /etc/conf.d/lxc

# Make sure LXD group is created
echo "LXD_OPTIONS=\" --group lxd\"" >> /etc/conf.d/lxd

# Sort out UID mappings
chmod -x /usr/bin/newuidmap
chmod -x /usr/bin/newgidmap

rc-update add lxc
rc-update add lxd
#rc-update add lxcfs
rc-update add dbus

reboot
