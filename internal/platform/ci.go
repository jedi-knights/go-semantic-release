package platform

import "os"

// CIEnvironment represents a detected CI/CD platform.
type CIEnvironment struct {
	Name     string
	Detected bool
	Branch   string
	Commit   string
	BuildURL string
}

// DetectCI checks common CI environment variables and returns the detected environment.
func DetectCI() CIEnvironment {
	detectors := []struct {
		name     string
		envVar   string
		branch   string
		commit   string
		buildURL string
	}{
		{
			name:     "GitHub Actions",
			envVar:   "GITHUB_ACTIONS",
			branch:   "GITHUB_REF_NAME",
			commit:   "GITHUB_SHA",
			buildURL: "GITHUB_SERVER_URL",
		},
		{
			name:     "GitLab CI",
			envVar:   "GITLAB_CI",
			branch:   "CI_COMMIT_BRANCH",
			commit:   "CI_COMMIT_SHA",
			buildURL: "CI_PIPELINE_URL",
		},
		{
			name:     "CircleCI",
			envVar:   "CIRCLECI",
			branch:   "CIRCLE_BRANCH",
			commit:   "CIRCLE_SHA1",
			buildURL: "CIRCLE_BUILD_URL",
		},
		{
			name:     "Travis CI",
			envVar:   "TRAVIS",
			branch:   "TRAVIS_BRANCH",
			commit:   "TRAVIS_COMMIT",
			buildURL: "TRAVIS_BUILD_WEB_URL",
		},
		{
			name:     "Jenkins",
			envVar:   "JENKINS_URL",
			branch:   "GIT_BRANCH",
			commit:   "GIT_COMMIT",
			buildURL: "BUILD_URL",
		},
		{
			name:     "Azure Pipelines",
			envVar:   "TF_BUILD",
			branch:   "BUILD_SOURCEBRANCH",
			commit:   "BUILD_SOURCEVERSION",
			buildURL: "SYSTEM_TEAMFOUNDATIONSERVERURI",
		},
		{
			name:     "Bitbucket Pipelines",
			envVar:   "BITBUCKET_BUILD_NUMBER",
			branch:   "BITBUCKET_BRANCH",
			commit:   "BITBUCKET_COMMIT",
			buildURL: "BITBUCKET_GIT_HTTP_ORIGIN",
		},
		{
			name:   "Generic CI",
			envVar: "CI",
		},
	}

	for _, d := range detectors {
		if os.Getenv(d.envVar) != "" {
			return CIEnvironment{
				Name:     d.name,
				Detected: true,
				Branch:   os.Getenv(d.branch),
				Commit:   os.Getenv(d.commit),
				BuildURL: os.Getenv(d.buildURL),
			}
		}
	}

	return CIEnvironment{Detected: false}
}

// IsCI returns true if running in a CI environment.
func IsCI() bool {
	return DetectCI().Detected
}
