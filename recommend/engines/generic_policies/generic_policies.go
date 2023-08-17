package genericpolicies

import (
	_ "embed" // need for embedding
	"fmt"
	"path/filepath"

	"regexp"
	"strings"

	"github.com/fatih/color"
	"github.com/kubearmor/kubearmor-client/recommend/common"
	"github.com/kubearmor/kubearmor-client/recommend/image"
	"github.com/kubearmor/kubearmor-client/recommend/report"
	log "github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"
)

const (
	org   = "kubearmor"
	repo  = "policy-templates"
	url   = "https://github.com/kubearmor/policy-templates/archive/refs/tags/"
	cache = ".cache/karmor/"
)

type GenericPolicy struct {
}

func (P GenericPolicy) Init() error {
	if _, err := DownloadAndUnzipRelease(); err != nil {
		return err
	}
	return nil
}

func (P GenericPolicy) Scan(img *image.ImageInfo, options common.Options, tags []string) error {
	if err := getPolicyFromImageInfo(img, options, tags); err != nil {
		log.WithError(err).Error("policy generation from image info failed")
	}
	return nil
}

func checkForSpec(spec string, fl []string) []string {
	var matches []string
	if !strings.HasSuffix(spec, "*") {
		spec = fmt.Sprintf("%s$", spec)
	}

	re := regexp.MustCompile(spec)
	for _, name := range fl {
		if re.Match([]byte(name)) {
			matches = append(matches, name)
		}
	}
	return matches
}

func matchTags(ms *common.MatchSpec, tags []string) bool {
	if len(tags) <= 0 {
		return true
	}
	for _, t := range tags {
		if slices.Contains(ms.Spec.Tags, t) {
			return true
		}
	}
	return false
}

func checkPreconditions(img *image.ImageInfo, ms *common.MatchSpec) bool {
	var matches []string
	for _, preCondition := range ms.Precondition {
		matches = append(matches, checkForSpec(filepath.Join(preCondition), img.FileList)...)
		if strings.Contains(preCondition, "OPTSCAN") {
			return true
		}
	}
	return len(matches) >= len(ms.Precondition)
}

func getPolicyFromImageInfo(img *image.ImageInfo, options common.Options, tags []string) error {
	if img.OS != "linux" {
		color.Red("non-linux platforms are not supported, yet.")
		return nil
	}

	idx := 0

	if err := report.ReportStart(img, options, CurrentVersion); err != nil {
		log.WithError(err).Error("report start failed")
		return err
	}
	var ms common.MatchSpec
	var err error

	ms, err = getNextRule(&idx)
	for ; err == nil; ms, err = getNextRule(&idx) {

		if !matchTags(&ms, tags) {
			continue
		}

		if !checkPreconditions(img, &ms) {
			continue
		}
		record := &report.Report{}
		img.WritePolicyFile(ms, record, options)
	}
	return nil
}
