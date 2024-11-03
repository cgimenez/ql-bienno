package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
)

func downloadNeeded(dst, _ string) (bool, error) {
	if !file_exists(dst) {
		return true, nil
		// response, err := http.Head(image.URL)
		// if err != nil {
		// 	return false, err
		// }
		// contentlength := response.ContentLength
		// file_info, _ := os.Stat(filename)
		// if file_info.Size() != contentlength {
		// 	return true, nil
		// }
	}
	return false, nil
}

func DownloadFile(dst, url string) error {
	if err := os.MkdirAll("downloads", 0755); err != nil {
		return fmt.Errorf("failed to create directory : %w", err)
	}

	dl_needed, err := downloadNeeded(dst, url)
	if err != nil {
		return err
	}

	if dl_needed {
		fmt.Printf("Downloading %s\n", url)
		out, err := os.Create(dst)
		if err != nil {
			return fmt.Errorf("create file : %w", err)
		}
		defer out.Close()

		resp, err := http.Get(url)
		if err != nil {
			return fmt.Errorf("GET url %s : %w", url, err)
		}
		defer resp.Body.Close()

		if _, err = io.Copy(out, resp.Body); err != nil {
			return fmt.Errorf("copy : %w", err)
		}
	}
	return nil
}
