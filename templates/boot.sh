# https://www.qemu.org/docs/master/system/invocation.html#hxtool-0
# https://wiki.qemu.org/Documentation/9psetup
# -virtfs local,path=/Users/chris/qemu_shared,mount_tag=mount_tag,security_model=passthrough \
# sudo mount -t 9p -o trans=virtio mount_tag ./host -oversion=9p2000.L
# -virtfs local,path=/Users/chris/qemu_shared,mount_tag=mount_tag,security_model=mapped-xattr,fmode=0666,dmode=0777 \
iface=$(route get default | grep interface | awk '{print $2}');
qemu-system-aarch64 -M virt \
{{ if not .Config.EnableVirtFS }}-run-with user={{ .Config.HostUser }}{{ end }} \
{{ if .Config.EnableVirtFS }}-virtfs local,path=./share,mount_tag=mount_tag,security_model=mapped-xattr,fmode=0666,dmode=0777 \{{ end }}
-accel hvf \
-smp {{ .Config.Smp }} -m {{ .Config.Mem }}G -cpu host \
-bios bios.fd \
-hda boot.qcow2 \
-nic vmnet-bridged,ifname=$iface,mac={{ .NetworkConfig.MacAddr }} \
-cdrom cidata.iso \
-qmp unix:./qemu-monitor,server,nowait \
-serial mon:stdio \
-nographic
