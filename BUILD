load("@io_bazel_rules_go//go:def.bzl", "go_prefix", "go_library", "go_binary")

go_prefix("github.com/lukegb/ibazel")

go_library(
    name = "depresolver",
    srcs = ["depresolver.go"],
)

go_binary(
    name = "ibazel",
    srcs = ["ibazel.go"],
    deps = [
        ":depresolver",
        "@org_golang_x_exp//:inotify",
    ],
)
