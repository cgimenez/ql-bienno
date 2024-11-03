# bienno-ql: A cheap QEMU frontend for macOS, targeting Debian VMs

> If it's already been done by someone else, do it at your own sauce.

-- <cite>Jean François Mélogiot (Maître saucier à la Maison Bourgeoise / Lyon), 1926</cite>

## Why ?

Because of the need for:

- Disposable, full-featured fresh Debian VMs that can be quickly available (in less than one minute) for Ansible experiments.
- A way to experiment with Cloud-Init scripts.
- A simple setup—though, even I had to dig a lot to find Internet resources
- Just why not ?

This project is intended for development and tweaking experiments. If you’re looking to run containers on macOS, consider better suited alternatives:

- [lima] <https://github.com/lima-vm/lima>
- [colima] <https://github.com/abiosoft/colima>

and for VMs

- [UTM] <https://github.com/utmapp/UTM>

## Requirements

- go
- qemu >= 9.1

## Untested on

- Intel Macs
- Alpine images (though attempted, it wasn’t ultimately needed)

## Where does it run ?

- Inside the cloned directory, there will be nothing left in you $HOME directory

## Security

Absolutely non secure, unless you change it to tailor your exceptations

- qemu start command need sudo but will -run-with your current host user privilege excepted if you're using virtfs
- root console password is : "root"
- if samba is enabled, then... meh
- The same SSH server keys are used (intentionally) for each instance creation
- have a look at Cloud-Init scripts in ./templates to check others security holes and fix them

## Installation

- `git clone`
- `cd ql-bienno`
- `./build.sh`

## Key points

- Instances are pingable/servicable from/to host & any other instances, an instance is like a real machines, so there not need to redirect ports or whatever.
- Source code is short (less than 1000 lines) : so you can tweak it to fullfill your needs.
- Fast, thanks to qemu native virtualization being **SO** fast !
- The setup may be somewhat rigid, but you can fork and modify it as needed.

## Make your own config

### Start from

`cp example_config/debian.yaml config/debian.yaml`

```YAML
image: debian-12-generic-arm64 # The stock image name
ip_address: 192.168.1.70 # The IP of the instance
user_name: debian # Username to be created
ssh_pub_key: .ssh/qemu.pub # The public SSH key for user access - see at the end of this document
smp: 2 # Number of CPUs (default: 2)
mem : 8 # Memory in GB (default: 8)
disk_size: 40 # Disk size in GB (default: 40)
samba: false # default false - If true, provides a Samba share (default user: user_name, password: user_name)
enable_virtfs: true # (default false) - if true you'll get a "share" directory under instances/my_instance_id and a "host" directory mounted on the guest, under ~
```

and edit `config/debian.yaml`

## Using instances

### Creating

```shell
./ql create foobar config=debian # config relates to config/debian.yaml
```

### Running

```shell
sudo ./ql start foobar # sudo is mandatory because of qemu -vmnet networking
```

or

```shell
sudo ./ql start foobar --verbose # you'll get the verbose stuff when the vm boots
```

Note : it's totally fine to

```shell
cd instances/foobar
sudo ./boot.sh
```

- you'll be able to login as root
- you'll still can use `sudo ql stop` or `ql status`

### Stopping

```shell
sudo ./ql stop foobar # sudo is mandatory because you used sudo to start the instance
```

### Destroying

```shell
./ql destroy foobar # will fail if instance is protected OR if enable_virtfs is true and the share folder is not empty
```

### Shell

```shell
./ql shell foobar # shortcut to ssh -i ...]
```

### Status

```shell
./ql status foobar
```

### Protecting (us against the Unknown)

```shell
./ql protect foobar
```

## Customize further

The best way to further customize instances is by forking this repo then editing and expanding

- the Cloud-Init templates located in `./templates`.
- the `instance_config.go` file

## Generate user ssh key (to be used in config.yaml)

```shell
ssh-keygen -t rsa -b 4096 -f ~/.ssh/qemu
```

or depending on you OS/settings

```shell
ssh-keygen -t rsa -b 4096 -f ~/.ssh/keys/qemu
```

## About the name

<cite>Such good times spent at the Red House in Bienno, Brescia, Italy.</cite>
