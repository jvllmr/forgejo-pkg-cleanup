package main

import (
	"fmt"
	"os"
	"regexp"
	"time"

	"codeberg.org/mvdkleijn/forgejo-sdk/forgejo/v3"
	"github.com/sethvargo/go-githubactions"
)

func RequiredInput(action *githubactions.Action, key string) (string, error) {
	val := action.GetInput(key)
	if val == "" {
		return val, fmt.Errorf("%s is a required input", key)
	}
	return val, nil
}

func ErrorOut(action *githubactions.Action, err error) {
	action.Errorf("fatal: %v", err)
	os.Exit(1)
}

func main() {
	action := githubactions.New()
	action.Noticef("Waiting for 3 seconds...")
	time.Sleep(3 * time.Second)
	action.Noticef("Starting to cleanup packages")
	url, err := RequiredInput(action, "instance")
	if err != nil {
		ErrorOut(action, err)
	}

	username, err := RequiredInput(action, "username")
	if err != nil {
		ErrorOut(action, err)
	}

	password, err := RequiredInput(action, "password")
	if err != nil {
		ErrorOut(action, err)
	}

	client, err := forgejo.NewClient(url, forgejo.SetBasicAuth(username, password))
	if err != nil {
		ErrorOut(action, err)
	}

	owner := action.GetInput("owner")
	if owner == "" {
		owner = username
	}
	packages, _, err := client.ListPackages(
		owner,
		forgejo.ListPackagesOptions{ListOptions: forgejo.ListOptions{Page: -1, PageSize: 0}},
	)
	if err != nil {
		ErrorOut(action, err)
	}

	pkgStr, err := RequiredInput(action, "package")
	if err != nil {
		ErrorOut(action, err)
	}

	nameRe, err := regexp.Compile(pkgStr)
	if err != nil {
		ErrorOut(action, err)
	}

	versionsStr, err := RequiredInput(action, "keepVersions")
	if err != nil {
		ErrorOut(action, err)
	}

	versionsRe, err := regexp.Compile(versionsStr)
	if err != nil {
		ErrorOut(action, err)
	}

	packageTypeStr, err := RequiredInput(action, "packageType")
	if err != nil {
		ErrorOut(action, err)
	}

	packageTypeRe, err := regexp.Compile(packageTypeStr)
	if err != nil {
		ErrorOut(action, err)
	}

	retentionStr, err := RequiredInput(action, "retention")
	if err != nil {
		ErrorOut(action, err)
	}

	retention, err := time.ParseDuration(retentionStr)
	if err != nil {
		ErrorOut(action, err)
	}

	deletionCount := 0
	for _, pkg := range packages {
		if !packageTypeRe.MatchString(pkg.Type) {
			action.Debugf("package type %s does not match %s", pkg.Type, packageTypeStr)
			continue
		}

		if !nameRe.MatchString(pkg.Name) {
			action.Debugf("package name %s does not match %s", pkg.Name, pkgStr)
			continue
		}

		if versionsRe.MatchString(pkg.Version) {
			action.Debugf(
				"keeping package version %s because it does match %s",
				pkg.Version,
				versionsStr,
			)
			continue
		}

		if time.Since(pkg.CreatedAt) >= retention {
			action.Debugf(
				"package created_at %s is inside retention span of %s",
				pkg.CreatedAt,
				retentionStr,
			)
			continue
		}

		_, err := client.DeletePackage(owner, pkg.Type, pkg.Name, pkg.Version)
		if err != nil {
			action.Errorf(
				"could not delete package %s %s (%s): %v",
				pkg.Name,
				pkg.Version,
				pkg.Type,
				err,
			)
		}
		action.Infof(
			"Deleted %s %s (%s)",
			pkg.Name,
			pkg.Version,
			pkg.Type,
		)
		deletionCount += 1
	}
	action.Infof("deleted %d packages", deletionCount)
}
