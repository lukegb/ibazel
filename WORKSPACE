git_repository(
    name = "io_bazel_rules_go",
    commit = "ae8ea32be1af991eef77d6347591dc8ba56c40a2",
    remote = "https://github.com/bazelbuild/rules_go.git",
)

new_git_repository(
    name = "org_golang_x_exp",
    build_file = "BUILD.org_golang_x_exp",
    commit = "325d5821a8d876702cbc82b93fec4ede356d498b",
    remote = "https://go.googlesource.com/exp",
)

load("@io_bazel_rules_go//go:def.bzl", "go_repositories")

go_repositories()
