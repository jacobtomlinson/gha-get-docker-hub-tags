package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/containers/image/v5/docker"
	"github.com/containers/image/v5/docker/reference"
	"github.com/containers/image/v5/types"
	"github.com/containers/skopeo/version"
	"github.com/coreos/go-semver/semver"
	"github.com/pkg/errors"
)

type dhtag struct {
	Name string `json:"name"`
}

type dhrepo struct {
	Count   int     `json:"count"`
	Results []dhtag `json:"results"`
}

var defaultUserAgent = "skopeo/" + version.Version

type globalOptions struct {
	debug              bool          // Enable debug output
	policyPath         string        // Path to a signature verification policy file
	insecurePolicy     bool          // Use an "allow everything" signature verification policy
	registriesDirPath  string        // Path to a "registries.d" registry configuration directory
	overrideArch       string        // Architecture to use for choosing images, instead of the runtime one
	overrideOS         string        // OS to use for choosing images, instead of the runtime one
	overrideVariant    string        // Architecture variant to use for choosing images, instead of the runtime one
	commandTimeout     time.Duration // Timeout for the command execution
	registriesConfPath string        // Path to the "registries.conf" file
	tmpDir             string        // Path to use for big temporary files
}

// Customized version of the alltransports.ParseImageName and docker.ParseReference that does not place a default tag in the reference
// Would really love to not have this, but needed to enforce tag-less and digest-less names
func parseDockerRepositoryReference(refString string) (types.ImageReference, error) {
	if !strings.HasPrefix(refString, docker.Transport.Name()+"://") {
		return nil, errors.Errorf("docker: image reference %s does not start with %s://", refString, docker.Transport.Name())
	}

	parts := strings.SplitN(refString, ":", 2)
	if len(parts) != 2 {
		return nil, errors.Errorf(`Invalid image name "%s", expected colon-separated transport:reference`, refString)
	}

	ref, err := reference.ParseNormalizedNamed(strings.TrimPrefix(parts[1], "//"))
	if err != nil {
		return nil, err
	}

	if !reference.IsNameOnly(ref) {
		return nil, errors.New(`No tag or digest allowed in reference`)
	}

	// Checks ok, now return a reference. This is a hack because the tag listing code expects a full image reference even though the tag is ignored
	return docker.NewReference(reference.TagNameOnly(ref))
}

func newSystemContext() *types.SystemContext {
	opts := globalOptions{}
	ctx := &types.SystemContext{
		RegistriesDirPath:        opts.registriesDirPath,
		ArchitectureChoice:       opts.overrideArch,
		OSChoice:                 opts.overrideOS,
		VariantChoice:            opts.overrideVariant,
		SystemRegistriesConfPath: opts.registriesConfPath,
		BigFilesTemporaryDir:     opts.tmpDir,
		DockerRegistryUserAgent:  defaultUserAgent,
	}
	return ctx
}

func main() {

	org := os.Getenv("INPUT_ORG")
	repo := os.Getenv("INPUT_REPO")

	ctx := context.Background()

	url := fmt.Sprintf(`docker://%s/%s`, org, repo)

	imgRef, err := parseDockerRepositoryReference(url)
	if err != nil {
		return
	}

	sys := newSystemContext()

	result, err := docker.GetRepositoryTags(ctx, sys, imgRef)
	if err != nil {
		return
	}

	var tags []*semver.Version
	for _, tag := range result {
		matched, _ := regexp.MatchString(`.*\..*\..*`, tag)
		if matched {
			tags = append(tags, semver.New(strings.Trim(tag, "vV")))
		}
	}

	if len(tags) == 0 {
		log.Fatal(fmt.Sprintf(`Unable to find tags for %s/%s`, org, repo))
	}

	semver.Sort(tags)
	fmt.Println(fmt.Sprintf(`::set-output name=tag::%s`, tags[len(tags)-1]))
}
