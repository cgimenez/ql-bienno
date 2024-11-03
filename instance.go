package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/user"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	probing "github.com/prometheus-community/pro-bing"
	"gopkg.in/yaml.v3"
)

type State int

const (
	Unknown State = iota
	Running
	Paused
	Stopped
)

type NetworkConfig struct {
	Iface               string
	SshUserPublicKey    string
	SshServerPublicKey  string
	SshServerPrivateKey string
	IPV6                string
	MacAddr             string
}

type Instance struct {
	ID             string
	ConfigFileName string
	Config         *InstanceConfig
	Dir            string
	ArchInfo       ArchInfo       // only used on create
	NetworkConfig  *NetworkConfig // only used on create
}

func buildInstance(id string, config_filename string) (*Instance, error) {
	conf := buildInstanceConfig()
	err := conf.Load(config_filename)
	if err != nil {
		return nil, err
	}
	return &Instance{
		ID:             id,
		ConfigFileName: config_filename,
		Config:         conf,
		Dir:            path.Join("instances", id),
		NetworkConfig:  &NetworkConfig{},
	}, nil
}

func (inst *Instance) exists() bool {
	return file_exists(inst.Dir)
}

// ----------------------------------------------------------------------------
// Put SSH keys into instance's NetworkConfig fields
// ----------------------------------------------------------------------------

func (inst *Instance) setupSshKeys() error {
	// First read the public user key
	home, _ := os.UserHomeDir()
	key_file := path.Join(home, inst.Config.SshPubKey)
	data, err := os.ReadFile(key_file)
	if err != nil {
		return fmt.Errorf("Reading user public ssh key %s : %w", key_file, err)
	}
	inst.NetworkConfig.SshUserPublicKey = string(data)

	// Then if server keys are missing, generate them
	if !file_exists(path.Join("keys", "default_server_key")) || !file_exists(path.Join("keys", "default_server_key.pub")) {
		fmt.Println("Generating server keys")
		_ = os.MkdirAll("keys", 0755)
		var publicKey, privateKey []byte
		if err := generateRSAKeys(&publicKey, &privateKey, 4096); err != nil {
			return err
		}
		if err := os.WriteFile("keys/default_server_key", privateKey, 0700); err != nil {
			return err
		}
		if err := os.WriteFile("keys/default_server_key.pub", publicKey, 0700); err != nil {
			return err
		}
	}

	// Then read the server private key
	key_file = path.Join("keys", "default_server_key")
	data, err = os.ReadFile(key_file)
	if err != nil {
		return fmt.Errorf("Reading user server ssh key %s : %w", key_file, err)
	}
	inst.NetworkConfig.SshServerPrivateKey = strings.Join(strings.Split(string(data), "\n"), "\n    ")

	// And read the server public key
	key_file = path.Join("keys", "default_server_key.pub")
	data, err = os.ReadFile(key_file)
	if err != nil {
		return fmt.Errorf("Reading user server public ssh key %s : %w", key_file, err)
	}
	inst.NetworkConfig.SshServerPublicKey = string(data)
	return nil
}

// ----------------------------------------------------------------------------
// Create a new instance - overwrite if forced and possible
// ----------------------------------------------------------------------------

func (inst *Instance) Create(force bool) error {
	if force {
		if err := inst.Destroy(); err != nil {
			return err
		}
	} else {
		if inst.exists() {
			return fmt.Errorf("Instance already exists")
		}
	}
	_ = os.MkdirAll(inst.Dir, 0755)

	// Download the base image
	image := buildStockImage(inst.Config.Image)
	inst.ArchInfo = image.ArchInfo
	err := image.download()
	if err != nil {
		return fmt.Errorf("Download failed : %w", err)
	}

	// And copy it to the instance folder then resize it
	src_image := path.Join("downloads", image.Name+".qcow2")
	dst_image := path.Join(inst.Dir, "boot.qcow2")
	if _, err = file_cp(src_image, dst_image); err != nil {
		return err
	}
	if err := inst.resizeBootDisk(); err != nil {
		return fmt.Errorf("Resize failed : %w", err)
	}

	// Copy the qemu firmware
	bios := fmt.Sprintf("/opt/local/share/qemu/edk2-%s-code.fd", qemuArch())
	if _, err = file_cp(bios, path.Join(inst.Dir, "bios.fd")); err != nil {
		return err
	}

	// Write the config file to the instance folder
	// Instead of copying, we Marshal because some Config keys are setup at creation time, eg. HostUser
	currentUser, err := user.Current()
	if err != nil {
		return err
	}
	inst.Config.HostUser = currentUser.Username
	out, err := yaml.Marshal(inst.Config)
	if err != nil {
		return err
	}

	file, err := os.OpenFile(path.Join(inst.Dir, "config.yaml"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.Write(out); err != nil {
		return err
	}
	// Get and setup SSH keys, both for user and server
	if err := inst.setupSshKeys(); err != nil {
		return err
	}

	// Network stuff
	if inst.ArchInfo.OS == "debian" {
		inst.NetworkConfig.Iface = "enp0s1"
	} else {
		inst.NetworkConfig.Iface = "eth0"
	}

	ipv6, err := IPv4ToIPv6(inst.Config.IpAddress)
	if err != nil {
		return nil
	}
	inst.NetworkConfig.IPV6 = ipv6

	mac, err := genMACAddr() // each instance get a random MAC addr
	if err != nil {
		return nil
	}
	inst.NetworkConfig.MacAddr = mac

	// Generate the boot.sh script
	if err := inst.genBootScript(); err != nil {
		return err
	}

	// Create the virtfs share directory, even if the instance does not use it
	if err := os.MkdirAll(path.Join(inst.Dir, "share"), 0755); err != nil {
		return err
	}

	// Build the Cloud Init .iso
	if err := inst.mk_iso(); err != nil {
		return err
	}
	fmt.Printf("Instance %s created\n", inst.ID)
	return nil
}

// ----------------------------------------------------------------------------
// Start an instance - with verbose you'll get the boot output
// ----------------------------------------------------------------------------

func (inst *Instance) Start(verbose bool) error {
	if state, _ := inst.state(); state != Stopped {
		return fmt.Errorf("Instance already running")
	}
	ipFree, err := inst.isIPFree()
	if err != nil {
		return err
	}
	if !ipFree {
		return fmt.Errorf("IP address %s is currently used by another host", inst.Config.IpAddress)
	}

	//if inst.Config.EnableVirtFS {
	// if err := inst.genBootScript(); err != nil {
	// 	return err
	// }
	//}
	absPath, _ := filepath.Abs(inst.Dir)
	cmd := exec.Command("/bin/sh", path.Join(absPath, "boot.sh"))
	cmd.Dir = absPath
	if verbose {
		cmd.Stdout = os.Stdout
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	fmt.Printf("Instance %s started\n", inst.ID)
	return nil
}

// ----------------------------------------------------------------------------
// Power down a running instance
// ----------------------------------------------------------------------------

func (inst *Instance) Stop() error {
	if state, _ := inst.state(); state != Running && state != Paused {
		return fmt.Errorf("Instance is not running")
	}
	socket, opError := inst.openSocket()
	if opError == nil {
		_, _ = socketCmd(socket, `{ "execute": "system_powerdown" }`)
		socket.Close()
		return nil
	}
	return fmt.Errorf("Can't stop instance %v", opError)
}

// ----------------------------------------------------------------------------
// Wipe an instance if not protected and share folder is empty
// ----------------------------------------------------------------------------

func (inst *Instance) Destroy() error {
	if inst.protected() {
		return fmt.Errorf("instance is protected")
	}
	if state, _ := inst.state(); state != Stopped {
		return fmt.Errorf("instance is running - stop it first")
	}

	share_folder := path.Join(inst.Dir, "share")
	empty, err := isFolderEmpty(share_folder)
	if err != nil {
		return err
	}
	if !empty {
		return fmt.Errorf("You can't destroy this instance because %s folder is not empty", share_folder)
	}

	err = os.RemoveAll(inst.Dir)
	if err != nil {
		return err
	}
	fmt.Printf("Instance %s destroyed\n", inst.ID)
	return nil
}

// ----------------------------------------------------------------------------
// Poor man instance protection
// ----------------------------------------------------------------------------

func (inst *Instance) Protect() error {
	file, err := os.Create(path.Join(inst.Dir, "protect"))
	if err != nil {
		return err
	}
	file.Close()
	fmt.Println("Instance is now protected")
	return nil
}

// ----------------------------------------------------------------------------
// Get instance status - Unknow might be caused by an underlying error
// ----------------------------------------------------------------------------

func (inst *Instance) Status() State {
	state, err := inst.state()
	if err != nil {
		fmt.Println(err)
	}
	switch state {
	case Stopped:
		fmt.Println("Stopped")
	case Running:
		fmt.Println("Running")
	case Paused:
		fmt.Println("Paused")
	default:
		fmt.Println("Unknow")
	}
	return state
}

// ----------------------------------------------------------------------------
// Open a shell - this is shortcut to ssh -i ...
// ----------------------------------------------------------------------------

func (inst *Instance) Shell() error {
	retry := 0
	for {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:22", inst.Config.IpAddress), 2000*time.Millisecond)
		if err != nil {
			if errors.Is(err, syscall.ECONNREFUSED) {
				break
			} else {
				fmt.Printf("Trying to connect %s@%s\n", inst.Config.UserName, inst.Config.IpAddress)
				time.Sleep(2000 * time.Millisecond)
				retry++
				if retry >= 5 {
					return fmt.Errorf("Instance seems be offline...")
				}
			}
		} else {
			conn.Close()
			break
		}
	}

	home, _ := os.UserHomeDir()
	ssh_key_file := strings.TrimSuffix(inst.Config.SshPubKey, ".pub")
	cmd := exec.Command("ssh", "-i", path.Join(home, ssh_key_file), fmt.Sprintf("%s@%s", inst.Config.UserName, inst.Config.IpAddress))
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 255 {
				return fmt.Errorf("the VM appears to be powered down or not responding")
			}
		}
		return fmt.Errorf("failed to run SSH command: %w", err)
	}
	return nil
}

// ----------------------------------------------------------------------------
// Resize
// ----------------------------------------------------------------------------

// ----------------------------------------------------------------------------
// Pseudo private methods
// ----------------------------------------------------------------------------

func (inst *Instance) protected() bool {
	return file_exists(path.Join(inst.Dir, "protect"))
}

func isFolderEmpty(path string) (bool, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false, err
	}
	return len(entries) == 0, nil
}

func (inst *Instance) genBootScript() error {
	boot_sh := path.Join(inst.Dir, "boot.sh")
	if err := inst.render_template("templates/boot.sh", boot_sh); err != nil {
		return err
	}
	return os.Chmod(boot_sh, 0755)
}

func (inst *Instance) isIPFree() (bool, error) {
	statistics, _ := inst.PingVM(1*time.Second, 1)
	if statistics == nil {
		return true, nil
	}
	return false, nil
}

func (inst *Instance) PingVM(timeout time.Duration, count int) (*probing.Statistics, error) {
	pinger, err := probing.NewPinger(inst.Config.IpAddress)
	if err != nil {
		return nil, err
	}
	pinger.Count = count
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := pinger.RunWithContext(ctx); err != nil {
		return nil, err
	}
	return pinger.Statistics(), nil
}

func (inst *Instance) resizeBootDisk() error {
	boot_file := path.Join(inst.Dir, "boot.qcow2")
	cmd := exec.Command("qemu-img", "info", boot_file, "--output=json")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	var result map[string]interface{}
	if err = json.Unmarshal([]byte(out), &result); err != nil {
		return nil
	}
	actual_size := int64(result["actual-size"].(float64))
	new_size := int64(inst.Config.DiskSize * 1024 * 1024 * 1024)
	if new_size > actual_size {
		cmd := exec.Command("qemu-img", "resize", boot_file, fmt.Sprintf("%dG", inst.Config.DiskSize))
		if err := cmd.Run(); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("Shrinking disk is not possible")
	}
	return nil
}

func (inst *Instance) state() (State, error) {
	socket, opError := inst.openSocket()
	if opError != nil {
		if sysErr, ok := opError.Err.(*os.SyscallError); ok {
			if sysErr.Err == syscall.EACCES || sysErr.Err == syscall.EPERM {
				return Running, nil
			}
		}
		if opError.Op == "dial" {
			return Stopped, nil
		}
	}

	res, err := socketCmd(socket, `{ "execute": "query-status" }`)
	if err == nil {
		socket.Close()
		var result map[string]interface{}
		err := json.Unmarshal([]byte(res), &result)
		if err != nil {
			return Unknown, fmt.Errorf("Parsing JSON %w", err)
		}
		returnData := result["return"].(map[string]interface{})
		switch returnData["status"].(string) {
		case "running":
			return Running, nil
		case "paused":
			return Paused, nil
		}
	}
	return Unknown, nil
}

func (inst *Instance) openSocket() (net.Conn, *net.OpError) {
	socket, err := net.Dial("unix", path.Join(inst.Dir, "qemu-monitor"))
	if err != nil {
		opError, _ := err.(*net.OpError)
		return nil, opError
	}
	_, _ = socketCmd(socket, `{ "execute": "qmp_capabilities" }`) // mandatory before any else command
	return socket, nil
}

func socketRead(socket net.Conn) (string, error) {
	time.Sleep(125 * time.Millisecond)
	res := make([]byte, 50000)
	n, err := socket.Read(res)
	if err == nil {
		return string(res[:n]), nil
	}
	return "", err
}

// Examples : https://gist.github.com/rgl/dc38c6875a53469fdebb2e9c0a220c6c
func socketCmd(socket net.Conn, cmd string) (string, error) {
	if _, err := socket.Write([]byte(cmd)); err != nil {
		return "", err
	}
	return socketRead(socket)
}
