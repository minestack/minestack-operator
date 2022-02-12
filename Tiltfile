# -*- mode: Python -*-

# set defaults
settings = {
    'preload_images_for_kind': True
}

# global settings
settings.update(read_json(
    "tilt-settings.json",
    default={},
))


def deploy_operator():
    custom_build(
        'controller',
        'docker build -t $EXPECTED_REF .',
        deps=[
            'bin/minestack-operator-linux-amd64',
            'Dockerfile'
        ],
    )

    yaml = str(kustomize("config/tilt", kustomize_bin="./bin/kustomize"))
    k8s_yaml(blob(yaml))


# Users may define their own Tilt customizations in tilt.d. This directory is excluded from git and these files will
# not be checked in to version control.
def include_user_tilt_files():
    user_tiltfiles = listdir("tilt.d")
    for f in user_tiltfiles:
        include(f)


##############################
# Actual work happens here
##############################
include_user_tilt_files()

deploy_operator()
