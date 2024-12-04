"""rpmtree repository rule

Create repositories that generate rpmtree rules and wrap up the entire
bazeldnf fetch + bazeldnf rpmtree process
"""

def _rpmtree_repository_impl(rctx):
    tool_path = rctx.path(rctx.attr.bazeldnf)
    rctx.watch_tree(tool_path.dirname)

    rctx.file("WORKSPACE")

    out = rctx.execute(
        [
            tool_path,
            "fetch",
            "--repofile",
            rctx.path(rctx.attr.repo_file),
        ],
    )

    if out.stderr and out.return_code != 0:
        fail("Error executing fetch: " + out.stderr)

    content = rctx.read(rctx.path(rctx.attr.rpms_file))
    rpms_file_json = json.decode(content)

    for rpm in rpms_file_json.get("rpms", []):
        build_file_path = "{}/BUILD".format(rpm["name"])

        rctx.file(
            build_file_path,
            content="""load("@bazeldnf//bazeldnf:defs.bzl", "rpmtree")
            """,
        )

        args = [
            tool_path,
            "rpmtree",
            "--configname",
            rctx.attr.configname,
            "--repofile",
            rctx.path(rctx.attr.repo_file),
            "--buildfile",
            build_file_path,
            "--name",
            "rpms",
            "--basesystem",
            rctx.attr.basesystem,
        ]

        for item in rpm.get("only_allow", []):
            args += ["--only-allow", item]

        for item in rpm.get("dont_allow", []):
            args += ["--force-ignore-with-dependencies", item]

        args += rpm["rpms"]
        out = rctx.execute(args)
        if out.stderr and out.return_code != 0:
            fail("Error executing rpmtree: " + out.stderr)

_rpmtree_attrs = {
    "repo_file": attr.label(
        doc = """\
Label of the YAML file that contains the DNF repo configuration.\
""",
        allow_single_file = [".yaml"],
    ),
    "rpms_file": attr.label(
    ),
    "bazeldnf": attr.label(),
    "configname": attr.string(),
    "basesystem": attr.string(),
}

rpmtree_repository = repository_rule(
    _rpmtree_repository_impl,
    attrs = _rpmtree_attrs,
)
