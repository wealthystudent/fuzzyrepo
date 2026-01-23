package main

// ---- Mock data helpers ----
func mockRepos() []RepoDTO {
	return []RepoDTO{
		{Name: "my-local-repo", ExistsLocal: true, Path: "/Users/REDACTED/code/my-local-repo"},
		{Name: "my-remote-repo", ExistsLocal: false, Path: "git@github.com:wealthystudent/my-remote-repo.git"},
		{Name: "org-service-api", ExistsLocal: false, Path: "git@github.com:myorg/service-api.git"},
		{Name: "org-service-api (local)", ExistsLocal: true, Path: "/Users/REDACTED/code/service-api"},
	}
}

func mockReposMore() []RepoDTO {
	return []RepoDTO{
		{Name: "testingrepo", ExistsLocal: false, Path: "git@github.com:wealthystudent/testingrepo.git"},
		{Name: "testingrepo2", ExistsLocal: false, Path: "git@github.com:wealthystudent/testingrepo2.git"},
	}
}
