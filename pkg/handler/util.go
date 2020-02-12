package handler

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/nektro/andesite/pkg/idata"
)

func processListingURL(pt map[string]string, upathS string) (string, string, error) {
	for k, v := range pt {
		if strings.HasPrefix(upathS, idata.Config.HTTPBase+k) {
			return v + "/", upathS[len(idata.Config.HTTPBase)+len(k):], nil
		}
	}
	return "", "", errors.New("not found")
}

func findRootForShareAccess(acc string) (string, string, error) {
	for k, v := range combine(idata.DataPathsPrv, idata.DataPathsPub) {
		if strings.HasPrefix(acc, "/"+k) {
			ap := strings.ReplaceAll(acc, "/"+k, v)
			if strings.HasSuffix(ap, "/") {
				return ap, "/", nil
			}
			ad := filepath.Dir(ap)
			return ad, ap[len(ad):], nil
		}
	}
	return "", "", errors.New("not found")
}

func combine(mps ...map[string]string) map[string]string {
	result := map[string]string{}
	for _, item := range mps {
		for k, v := range item {
			result[k] = v
		}
	}
	return result
}
