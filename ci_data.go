package main

import (
	"fmt"
	"os"
	"os/exec"
	"path"
)

// ----------------------------------------------------------------------------
// Build the Cloud Init ISO image
// ----------------------------------------------------------------------------

func (inst *Instance) mk_iso() error {
	tmp := path.Join("tmp", "iso_data")
	if err := os.RemoveAll(tmp); err != nil {
		return err
	}

	if err := os.MkdirAll(tmp, 0755); err != nil {
		return err
	}

	iso_image := path.Join("tmp", "cidata.iso")
	_ = os.Remove(iso_image)

	template_files := []string{
		"meta-data",
		"network-config",
		"user-data",
	}

	for _, template_file := range template_files {
		if err := inst.render_template(path.Join("templates", template_file), path.Join(tmp, template_file)); err != nil {
			return err
		}
	}

	cmd := exec.Command("hdiutil", "makehybrid", "-ov", "-iso", "-joliet", "-default-volume-name", "cidata", "-o", "tmp/cidata.iso", "tmp/iso_data")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Cmd hdutil failed with [%s] %w", string(out), err)
	}

	if _, err = file_cp("tmp/cidata.iso", path.Join(inst.Dir, "cidata.iso")); err != nil {
		return fmt.Errorf("Copying iso %w", err)
	}
	return nil
}
